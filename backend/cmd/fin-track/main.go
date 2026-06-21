package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mmarci96/fin-track/internal/config"
	"github.com/mmarci96/fin-track/internal/repository"
	"github.com/mmarci96/fin-track/internal/router"
	"github.com/mmarci96/fin-track/internal/service/ollama"
	"github.com/mmarci96/fin-track/pkg/logger"
)

func main() {
	cfg := config.NewConfig()

	if err := logger.Init(cfg.LogLevel, cfg.RuntimeEnv); err != nil {
		// The logger isn't up yet, so report to stderr and bail.
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}
	// Flush buffered logs on the way out.
	defer logger.Sync()

	if err := run(cfg); err != nil {
		logger.Log.Error("server exited with error", "error", err)
		// Sync runs via the deferred call before the process ends.
		os.Exit(1)
	}
}

// run wires up dependencies and serves until a shutdown signal arrives,
// returning an error instead of panicking so main can log and flush.
func run(cfg *config.AppConfig) error {
	db, err := repository.NewDatabase(cfg.DatabaseURL())
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.DB.Close()

	if err := db.EnsureDefaultUser(cfg.DefaultUserID); err != nil {
		return fmt.Errorf("ensure default user: %w", err)
	}

	ollamaSvc := ollama.NewOllamaService(*cfg, logger.Log)

	engine := router.SetupRouter(db, cfg, ollamaSvc)

	srv := &http.Server{
		Addr:    cfg.Host + ":" + cfg.Port,
		Handler: engine,
	}

	// Listen for interrupt/terminate so in-flight requests can drain.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		logger.Log.Info("server starting", "addr", srv.Addr, "env", cfg.RuntimeEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("listen and serve: %w", err)
	case <-ctx.Done():
		logger.Log.Info("shutdown signal received, draining requests")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	logger.Log.Info("server stopped cleanly")
	return nil
}
