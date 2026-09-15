package observability

import (
	"context"
	"log/slog"
	"os"

	"github.com/iheanyi-dev/ecommerce-backend/gateway/application/ports"
)

// Logger is the Gateway's slog-backed structured logger.
type Logger struct {
	logger *slog.Logger
}

// NewLogger creates the Gateway structured logger.
//
// JSON output is used so logs can be consumed by centralized logging
// infrastructure later without changing the application-facing logger port.
func NewLogger() *Logger {
	return &Logger{
		logger: slog.New(
			slog.NewJSONHandler(
				os.Stdout,
				&slog.HandlerOptions{
					Level: slog.LevelInfo,
				},
			),
		),
	}
}

// Info records an informational application event.
func (l *Logger) Info(
	ctx context.Context,
	message string,
	fields ...ports.Field,
) {
	l.logger.Log(
		ctx,
		slog.LevelInfo,
		message,
		toSlogAttrs(fields)...,
	)
}

// Error records an error application event.
func (l *Logger) Error(
	ctx context.Context,
	message string,
	fields ...ports.Field,
) {
	l.logger.Log(
		ctx,
		slog.LevelError,
		message,
		toSlogAttrs(fields)...,
	)
}

func toSlogAttrs(fields []ports.Field) []any {
	attrs := make([]any, 0, len(fields)*2)

	for _, field := range fields {
		attrs = append(attrs, slog.Any(field.Key, field.Value))
	}

	return attrs
}
