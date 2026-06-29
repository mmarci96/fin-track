// Command auth-service is a small, stateless authentication microservice that
// fronts the Traefik-controlled services. It authenticates users against its
// own dedicated database, issues HS256 JWTs (session cookie + Bearer), and the
// Traefik `traefikauth` plugin verifies those tokens locally and injects the
// resolved fin-track user id as the X-User-ID header.
//
// Usage:
//
//	auth-service                                  run the HTTP server
//	auth-service -createuser -email a@b.c \
//	    -password secret -app-user-id 1           create/update a user, then exit
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Port           string        `env:"AUTH_PORT"`
	DbHost         string        `env:"DB_HOST"`
	DbPort         string        `env:"DB_PORT"`
	DbName         string        `env:"DB_NAME"`
	DbUser         string        `env:"DB_USER"`
	DbPassword     string        `env:"DB_PASSWORD"`
	DbPasswordPath string        `env:"DB_PASSWORD_PATH"`
	JWTSecretEnv   string        `env:"JWT_SECRET"`      // inline fallback for one-shot tooling
	JWTSecretPath  string        `env:"JWT_SECRET_PATH"` // docker/swarm secret file
	JWTSecret      []byte        // resolved in loadConfig (never parsed from env directly)
	TokenTTL       time.Duration `env:"TOKEN_TTL"`
	CookieSecure   bool          `env:"COOKIE_SECURE"`
	CookieDomain   string        `env:"COOKIE_DOMAIN"`
}

func (c *Config) databaseUrl() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.DbUser,
		c.DbPassword,
		c.DbHost,
		c.DbPort,
		c.DbName,
	)
}

func loadConfig() Config {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		panic(err)
	}

	// JWT secret: prefer the mounted secret file (docker/swarm secret), and fall
	// back to JWT_SECRET for one-shot tooling like `make create-user`. TrimSpace
	// keeps the bytes identical to the traefikauth plugin, which reads the same
	// file — a stray trailing newline must not change the HMAC.
	switch {
	case cfg.JWTSecretPath != "":
		secret, err := loadFromFile(cfg.JWTSecretPath)
		if err != nil {
			panic(err)
		}
		cfg.JWTSecret = bytes.TrimSpace(secret)
	case cfg.JWTSecretEnv != "":
		cfg.JWTSecret = []byte(strings.TrimSpace(cfg.JWTSecretEnv))
	default:
		panic("auth-service: no JWT secret (set JWT_SECRET_PATH or JWT_SECRET)")
	}

	// DB password: prefer the mounted secret file, else the DB_PASSWORD env.
	if cfg.DbPasswordPath != "" {
		pw, err := loadFromFile(cfg.DbPasswordPath)
		if err != nil {
			panic(err)
		}
		cfg.DbPassword = strings.TrimSpace(string(pw))
	}

	return cfg
}

func main() {
	var (
		createUser = flag.Bool("createuser", false, "create or update a user, then exit")
		email      = flag.String("email", "", "user email (with -createuser)")
		password   = flag.String("password", "", "user password (with -createuser)")
		appUserID  = flag.Int("app-user-id", 0, "fin-track users.id to inject as X-User-ID (with -createuser)")
	)
	flag.Parse()

	cfg := loadConfig()

	store, err := openStore(cfg.databaseUrl())
	if err != nil {
		log.Fatalf("auth store: %v", err)
	}
	defer store.Close()

	if *createUser {
		if *email == "" || *password == "" || *appUserID <= 0 {
			log.Fatal("-createuser requires -email, -password and a positive -app-user-id")
		}
		if err := store.CreateUser(*email, *password, *appUserID); err != nil {
			log.Fatalf("create user: %v", err)
		}
		log.Printf("user %q ready (app_user_id=%d)", *email, *appUserID)
		return
	}

	srv := newServer(store, cfg)
	addr := ":" + cfg.Port
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           srv.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("auth-service listening on %s (cookie_secure=%v)", addr, cfg.CookieSecure)
	if err := httpSrv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
