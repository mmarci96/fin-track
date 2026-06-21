package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"

	slogzap "github.com/samber/slog-zap/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log is the package-level slog facade used across the app. It is also wired
// in as slog's default logger by Init.
var Log *slog.Logger

// zapBase keeps a handle to the underlying zap logger so Sync can flush the
// buffered entries on shutdown. Without this, buffered logs are lost on crash.
var zapBase *zap.Logger

// Init configures the global logger. In development environments it uses a
// human-friendly console encoder; everywhere else it emits structured JSON.
func Init(level, runtimeEnv string) error {
	zapCfg := zap.NewProductionConfig()

	if isDevelopment(runtimeEnv) {
		zapCfg = zap.NewDevelopmentConfig()
	}

	zapCfg.Level = zap.NewAtomicLevelAt(parseLevel(level))

	zapLogger, err := zapCfg.Build()
	if err != nil {
		return err
	}

	zapBase = zapLogger

	handler := slogzap.Option{
		Level:  slog.LevelDebug,
		Logger: zapLogger,
	}.NewZapHandler()

	Log = slog.New(handler)

	slog.SetDefault(Log)

	return nil
}

// Sync flushes any buffered log entries. Call it via defer in main so logs are
// not lost when the process exits.
func Sync() {
	if zapBase == nil {
		return
	}

	// Sync errors on stdout/stderr are expected on some platforms and safe to
	// ignore.
	_ = zapBase.Sync()
}

// isDevelopment treats the common dev-ish runtime env values the same so the
// console encoder reliably activates. The config default ("dev") used to fall
// through the old "development"-only check.
func isDevelopment(runtimeEnv string) bool {
	switch strings.ToLower(strings.TrimSpace(runtimeEnv)) {
	case "dev", "development", "local":
		return true
	default:
		return false
	}
}

func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return zapcore.DebugLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// With returns a child logger with the given attributes. It is safe to call
// before Init (falls back to a stderr text logger).
func With(args ...any) *slog.Logger {
	if Log == nil {
		return slog.New(slog.NewTextHandler(os.Stderr, nil)).With(args...)
	}

	return Log.With(args...)
}

type loggerKey struct{}

// FromContext returns the request-scoped logger stored in ctx, or the global
// logger if none is present.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return l
	}

	if Log != nil {
		return Log
	}

	return slog.New(slog.NewTextHandler(os.Stderr, nil))
}

// ToContext stores a logger in ctx so downstream code can retrieve it with
// FromContext.
func ToContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, l)
}
