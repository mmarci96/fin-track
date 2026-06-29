package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/caarlos0/env/v11"
)

type AppConfig struct {
	// App Config
	Host string `env:"HOST" envDefault:"localhost"`
	Port string `env:"PORT" envDefault:"8080"`

	// Ollama connection
	OllamaHost  string `env:"OLLAMA_HOST"  envDefault:"localhost"`
	OllamaPort  string `env:"OLLAMA_PORT" envDefault:"11434"`
	OllamaModel string `env:"OLLAMA_MODEL" envDefault:"llama3.2"`

	// Logging values
	LogLevel   string `env:"LOG_LEVEL" envDefault:"DEBUG"`
	RuntimeEnv string `env:"RUNTIME_ENV" envDefault:"dev"`

	// Default user assigned to receipts when no X-User-ID header is present and
	// RequireUserID is false (local dev hitting the backend directly).
	DefaultUserID int `env:"DEFAULT_USER_ID" envDefault:"1"`

	// RequireUserID rejects requests that arrive without a valid X-User-ID
	// header instead of falling back to DefaultUserID. Set true behind the auth
	// edge (UAT/prod) so the backend never invents an identity.
	RequireUserID bool `env:"REQUIRE_USER_ID" envDefault:"false"`

	// ImageStoreDir is where the debug upload endpoint persists original receipt
	// images (backed by a docker volume in deployment) so they can be viewed
	// next to their transcript. The dataset this builds up feeds the recognition
	// flywheel (known-products, OCR error rules).
	ImageStoreDir string `env:"IMAGE_STORE_DIR" envDefault:"./data/receipt-images"`

	// DB connection and values
	DbHost     string `env:"DB_HOST"`
	DbPort     string `env:"DB_PORT"`
	DbPassword string `env:"DB_PASSWORD"`
	// DbPasswordPath, when set, is a file (e.g. a mounted Docker secret) the DB
	// password is read from; it takes precedence over DB_PASSWORD.
	DbPasswordPath string `env:"DB_PASSWORD_PATH"`
	DbName         string `env:"DB_NAME"`
	DbUser         string `env:"DB_USER"`
}

func (c *AppConfig) DatabaseURL() string {
	// Build via net/url so credentials are percent-encoded. Secret-derived
	// passwords are base64 and routinely contain '/', '+' and '=', which break a
	// naive fmt.Sprintf DSN (the '/' gets parsed as the path, etc.).
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.DbUser, c.DbPassword),
		Host:     fmt.Sprintf("%s:%s", c.DbHost, c.DbPort),
		Path:     "/" + c.DbName,
		RawQuery: "sslmode=disable",
	}
	return u.String()
}

func NewConfig(options ...func(*AppConfig)) *AppConfig {
	cfg, err := env.ParseAs[AppConfig]()
	if err != nil {
		panic(err)
	}

	// Prefer the mounted secret file (Docker/Swarm secret) over DB_PASSWORD, so
	// the password never has to travel as a plaintext env var. dev-start, which
	// sets DB_PASSWORD and no path, is unaffected.
	if cfg.DbPasswordPath != "" {
		b, err := os.ReadFile(cfg.DbPasswordPath)
		if err != nil {
			panic(fmt.Errorf("read DB_PASSWORD_PATH %q: %w", cfg.DbPasswordPath, err))
		}
		cfg.DbPassword = strings.TrimSpace(string(b))
	}

	for _, o := range options {
		o(&cfg)
	}
	return &cfg
}

func WithHost(host string) func(*AppConfig) {
	return func(ac *AppConfig) {
		ac.Host = host
	}
}

func WithPort(port string) func(*AppConfig) {
	return func(ac *AppConfig) {
		ac.Port = port
	}

}
