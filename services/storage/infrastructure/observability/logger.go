package observability

import (
	"context"
	"errors"
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
)

var (
	// ErrNilWriter indicates that the logger was created without a valid
	// destination for its structured log output.
	ErrNilWriter = errors.New("logger writer cannot be nil")
)

// Logger is the Infrastructure implementation of the application-level
// ports.Logger contract.
//
// The application layer depends only on ports.Logger. This type owns the
// concrete Zap implementation and keeps the logging library isolated inside
// Infrastructure.
type Logger struct {
	logger *zap.Logger
}

// NewLogger creates a structured JSON logger that writes to the supplied
// destination.
//
// A writer is supplied instead of hard-coding os.Stdout so production can
// write to standard output while tests can capture log output in memory.
func NewLogger(writer io.Writer) (*Logger, error) {
	if writer == nil {
		return nil, ErrNilWriter
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(writer),
		zap.InfoLevel,
	)

	return &Logger{
		logger: zap.New(core),
	}, nil
}

// NewProductionLogger creates the Storage service production logger.
func NewProductionLogger() (*Logger, error) {
	return NewLogger(os.Stdout)
}

// Log records a structured application event.
//
// Only fields explicitly defined by ports.LogEvent are emitted. Object
// contents, credentials, tokens, and other secrets are never part of the
// logging contract.
func (l *Logger) Log(
	ctx context.Context,
	event ports.LogEvent,
) error {
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
