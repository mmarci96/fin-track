// Package traefikauth is a Traefik middleware plugin that gate-keeps the
// services behind the edge. It verifies a stateless HS256 JWT (issued by the
// auth-service) taken from the `session` cookie or a Bearer header, and on
// success injects the authenticated user's id as the X-User-ID header that the
// fin-track gin backend already reads. Verification is local (no per-request
// call to the auth-service), so the plugin only needs the shared JWT secret.
//
// The plugin runs under Yaegi, so it deliberately uses the standard library
// only (no third-party JWT/crypto imports).
package traefikauth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds plugin settings supplied from the Traefik dynamic config.
type Config struct {
	// JWTSecret is the shared HS256 secret used to verify tokens. It is supplied
	// either inline (Value) or, preferably, from a mounted file (UsersFile) such
	// as a Docker secret.
	JWTSecret SecretSource `json:"jwtSecret,omitempty"`
	// LoginURL is the absolute URL browsers are redirected to when unauthenticated.
	LoginURL string `json:"loginURL,omitempty"`
}

// SecretSource supplies a secret either inline or from a file on disk. UsersFile
// takes precedence so the real secret stays out of the dynamic config.
type SecretSource struct {
	UsersFile string `json:"usersFile,omitempty"`
	Value     string `json:"value,omitempty"`
}

// CreateConfig initializes default config.
func CreateConfig() *Config {
	return &Config{}
}

// TraefikAuth middleware.
type TraefikAuth struct {
	next      http.Handler
	jwtSecret string
	loginURL  string
	name      string
}

// New creates a new middleware instance.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	secret := config.JWTSecret.Value
	if config.JWTSecret.UsersFile != "" {
		b, err := os.ReadFile(config.JWTSecret.UsersFile)
		if err != nil {
			return nil, fmt.Errorf("traefikauth: read jwt secret %q: %w", config.JWTSecret.UsersFile, err)
		}
		secret = string(b)
	}
	// TrimSpace must match auth-service so both sides HMAC over identical bytes.
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, fmt.Errorf("traefikauth: jwt secret is empty (set jwtSecret.usersFile or jwtSecret.value)")
	}
	return &TraefikAuth{
		next:      next,
		jwtSecret: secret,
		loginURL:  config.LoginURL,
		name:      name,
	}, nil
}

// userIDHeader matches httpx.UserIDHeader in the fin-track backend.
const userIDHeader = "X-User-ID"

func (a *TraefikAuth) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Never trust a client-supplied identity header; only this middleware may
	// set it, derived from a verified token.
	req.Header.Del(userIDHeader)

	// The auth-service endpoints are unauthenticated by nature.
	switch req.URL.Path {
	case "/login", "/logout", "/verify", "/healthz":
		a.next.ServeHTTP(rw, req)
		return
	}

	uid, ok := a.authenticate(req)
	if ok {
		req.Header.Set(userIDHeader, strconv.Itoa(uid))
		a.next.ServeHTTP(rw, req)
		return
	}

	if isBrowser(req) && a.loginURL != "" {
		http.Redirect(rw, req, a.loginURL+"?next="+req.URL.RequestURI(), http.StatusSeeOther)
		return
	}
	http.Error(rw, "Unauthorized", http.StatusUnauthorized)
}

// authenticate extracts and verifies the token, returning the user id claim.
func (a *TraefikAuth) authenticate(req *http.Request) (int, bool) {
	token := tokenFromRequest(req)
	if token == "" {
		return 0, false
	}
	return verifyJWT(a.jwtSecret, token)
}

func tokenFromRequest(req *http.Request) string {
	if c, err := req.Cookie("session"); err == nil && c.Value != "" {
		return c.Value
	}
	if h := req.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}

// verifyJWT validates an HS256 token and returns its `uid` claim. It mirrors the
// auth-service implementation exactly so the two stay byte-compatible.
func verifyJWT(secret, token string) (int, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, false
	}
	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return 0, false
	}

	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, false
	}
	var claims struct {
		UID int   `json:"uid"`
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(raw, &claims); err != nil {
		return 0, false
	}
	if claims.Exp != 0 && time.Now().Unix() >= claims.Exp {
		return 0, false
	}
	if claims.UID <= 0 {
		return 0, false
	}
	return claims.UID, true
}

// isBrowser is a coarse check: API clients (curl, the frontend's fetch with a
// Bearer token) get a 401; humans get redirected to the login page.
func isBrowser(req *http.Request) bool {
	ua := req.Header.Get("User-Agent")
	return ua != "" && !strings.HasPrefix(ua, "curl")
}
