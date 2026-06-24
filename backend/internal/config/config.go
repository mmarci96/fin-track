package config

import (
	"fmt"

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

	// DB connection and values
	DbHost     string `env:"DB_HOST"`
	DbPort     string `env:"DB_PORT"`
	DbPassword string `env:"DB_PASSWORD"`
	DbName     string `env:"DB_NAME"`
	DbUser     string `env:"DB_USER"`
}

func (c *AppConfig) DatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.DbUser,
		c.DbPassword,
		c.DbHost,
		c.DbPort,
		c.DbName,
	)
}

func NewConfig(options ...func(*AppConfig)) *AppConfig {
	var cfg AppConfig
	err := env.Parse(&cfg)
	if err != nil {
		panic(err)
	}
	cfg, err = env.ParseAs[AppConfig]()
	if err != nil {
		panic(err)
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
