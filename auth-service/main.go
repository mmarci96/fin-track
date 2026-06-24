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
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Port         string        `env:"AUTH_PORT"`
	DbHost       string        `env:"DB_HOST"`
	DbPort       string        `env:"DB_PORT"`
	DbPassword   string        `env:"DB_PASSWORD"`
	DbName       string        `env:"DB_NAME"`
	DbUser       string        `env:"DB_USER"`
	JWTSecret    string        `env:"JWT_SECRET"`
	TokenTTL     time.Duration `env:"TOKEN_TTL"`
	CookieSecure bool          `env:"COOKIE_SECURE"`
	CookieDomain string        `env:"COOKIE_DOMAIN"`
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
	var cfg Config
	err := env.Parse(&cfg)
	if err != nil {
		panic(err)
	}
	cfg, err = env.ParseAs[Config]()
	if err != nil {
		panic(err)
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
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

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
