package main

import (
	"embed"
	"errors"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//go:embed templates/*.html
var templatesFS embed.FS

var loginTmpl = template.Must(template.ParseFS(templatesFS, "templates/login.html"))

const cookieName = "session"

// Server holds the auth HTTP handlers.
type Server struct {
	store *Store
	cfg   Config
}

func newServer(store *Store, cfg Config) *Server {
	return &Server{store: store, cfg: cfg}
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /login", s.loginPage)
	mux.HandleFunc("POST /login", s.login)
	mux.HandleFunc("GET /verify", s.verify)
	mux.HandleFunc("POST /logout", s.logout)
	mux.HandleFunc("GET /logout", s.logout)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return mux
}

// safeNext only allows same-site relative redirect targets, never absolute URLs
// (which would enable open-redirect phishing).
func safeNext(next string) string {
	if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") {
		return "/"
	}
	return next
}

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	render(w, "", safeNext(r.URL.Query().Get("next")))
}

func render(w http.ResponseWriter, errMsg, next string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = loginTmpl.Execute(w, map[string]string{"Error": errMsg, "Next": next})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	email := strings.TrimSpace(r.PostForm.Get("email"))
	password := r.PostForm.Get("password")
	next := safeNext(r.PostForm.Get("next"))

	user, err := s.store.Authenticate(email, password)
	if err != nil {
		if !errors.Is(err, errInvalidCredentials) {
			log.Printf("auth error for %q: %v", email, err)
		}
		w.WriteHeader(http.StatusUnauthorized)
		render(w, "Invalid email or password.", next)
		return
	}

	token, err := s.issueToken(user)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, s.sessionCookie(token, s.cfg.TokenTTL))

	// API clients can ask for the raw token instead of a redirect.
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"` + token + `"}`))
		return
	}
	http.Redirect(w, r, next, http.StatusSeeOther)
}

// verify validates the session for forward-auth consumers and echoes the
// resolved identity back as headers. The Traefik plugin verifies tokens locally
// (stateless JWT) so this endpoint is mainly for other services / debugging.
func (s *Server) verify(w http.ResponseWriter, r *http.Request) {
	token := tokenFromRequest(r)
	if token == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	claims, err := verifyJWT(s.cfg.JWTSecret, token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.Header().Set("X-User-ID", strconv.Itoa(claims.UID))
	w.Header().Set("X-Auth-User", claims.Sub)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	c := s.sessionCookie("", -time.Hour)
	c.MaxAge = -1
	http.SetCookie(w, c)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) issueToken(u *User) (string, error) {
	now := time.Now()
	return signJWT(s.cfg.JWTSecret, Claims{
		Sub: u.Email,
		UID: u.AppUserID,
		Iat: now.Unix(),
		Exp: now.Add(s.cfg.TokenTTL).Unix(),
	})
}

func (s *Server) sessionCookie(value string, ttl time.Duration) *http.Cookie {
	return &http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		Domain:   s.cfg.CookieDomain,
		Expires:  time.Now().Add(ttl),
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
}

// tokenFromRequest pulls a JWT from the session cookie or a Bearer header.
func tokenFromRequest(r *http.Request) string {
	if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
		return c.Value
	}
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}
