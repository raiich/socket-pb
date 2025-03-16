package log

import (
	"context"
	"log/slog"
	"os"

	"github.com/raiich/socket-pb/lib/log"
)

var defaultLogger log.Logger = slog.New(&log.Handler{
	Handler: slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}),
})

func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

func DebugContext(ctx context.Context, msg string, args ...any) {
	defaultLogger.DebugContext(ctx, msg, args...)
}

func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

func InfoContext(ctx context.Context, msg string, args ...any) {
	defaultLogger.InfoContext(ctx, msg, args...)
}

func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

func WarnContext(ctx context.Context, msg string, args ...any) {
	defaultLogger.WarnContext(ctx, msg, args...)
}

func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

func ErrorContext(ctx context.Context, msg string, args ...any) {
	defaultLogger.ErrorContext(ctx, msg, args...)
}

func OnError(err error, args ...any) {
	if err != nil {
		defaultLogger.Error("unexpected error", append(args, "error", err)...)
	}
}

func OnErrorContext(ctx context.Context, err error, args ...any) {
	if err != nil {
		defaultLogger.ErrorContext(ctx, "unexpected error", append(args, "error", err)...)
	}
}
