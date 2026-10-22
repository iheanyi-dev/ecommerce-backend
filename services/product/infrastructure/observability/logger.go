package observability

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// ErrNilWriter indicates that the logger has no output destination.
	ErrNilWriter = errors.New("logger writer cannot be nil")
)

// Logger adapts Zap to the Product application logger port.
type Logger struct {
	logger *zap.Logger
}

var _ ports.Logger = (*Logger)(nil)

// NewLogger creates a JSON logger writing to the supplied destination.
func NewLogger(writer io.Writer) (*Logger, error) {
	if writer == nil {
		return nil, ErrNilWriter
	}

	config := zap.NewProductionEncoderConfig()
	config.TimeKey = "timestamp"

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(config),
		zapcore.AddSync(writer),
		zap.InfoLevel,
	)

	return &Logger{logger: zap.New(core)}, nil
}

// NewProductionLogger creates a logger writing structured JSON to stdout.
func NewProductionLogger() (*Logger, error) {
	return NewLogger(os.Stdout)
}

// Log writes only fields explicitly allowed by the application port.
func (l *Logger) Log(_ context.Context, event ports.LogEvent) error {
	if l == nil || l.logger == nil {
		return errors.New("logger is not initialized")
	}

	fields := []zap.Field{
		zap.String("event", event.Event),
		zap.String("operation", event.Operation),
	}

	if event.FailureCategory != "" {
		fields = append(
			fields,
			zap.String("failure_category", event.FailureCategory),
		)
	}

	l.logger.Info(event.Event, fields...)
	return nil
}
