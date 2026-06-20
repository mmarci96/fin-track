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

var Log *slog.Logger

func Init(level, runtimeEnv string) error {
	zapCfg := zap.NewProductionConfig()

	if strings.ToLower(runtimeEnv) == "development" {
		zapCfg = zap.NewDevelopmentConfig()
	}

	zapCfg.Level = zap.NewAtomicLevelAt(parseLevel(level))

	zapLogger, err := zapCfg.Build()
	if err != nil {
		return err
	}

	handler := slogzap.Option{
		Level:  slog.LevelDebug,
		Logger: zapLogger,
	}.NewZapHandler()

	Log = slog.New(handler)

	slog.SetDefault(Log)

	return nil
}

func Sync() {
	if Log == nil {
		return
	}

	if z, ok := extractZap(); ok {
		_ = z.Sync()
	}
}

func extractZap() (*zap.Logger, bool) {
	return nil, false
}

func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func With(args ...any) *slog.Logger {
	if Log == nil {
		return slog.New(slog.NewTextHandler(os.Stdout, nil)).With(args...)
	}

	return Log.With(args...)
}

func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return l
	}

	return Log
}

type loggerKey struct{}

func ToContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, l)
}
