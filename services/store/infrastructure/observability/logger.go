package observability

import (
	"context"
	"errors"
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
)

var (
	// ErrNilWriter indicates that the logger was created without a valid
	// destination for its structured log output.
	ErrNilWriter = errors.New("logger writer cannot be nil")
)

// Logger is the Infrastructure implementation of the Store
// application-level ports.Logger contract.
//
// The application layer depends only on ports.Logger. Zap remains isolated
// inside Infrastructure.
type Logger struct {
	logger *zap.Logger
}

// Compile-time contract verification.
var _ ports.Logger = (*Logger)(nil)

// NewLogger creates a structured JSON logger that writes to the supplied
// destination.
//
// A writer is injected so production can use stdout while tests can capture
// structured output without depending on the process environment.
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

// NewProductionLogger creates the production Store logger.
//
// Container runtimes can collect stdout without the service managing log
// files itself.
func NewProductionLogger() (*Logger, error) {
	return NewLogger(os.Stdout)
}

// Log records a safe structured Store application event.
//
// Only fields explicitly defined by ports.LogEvent are emitted. This keeps
// credentials, tokens, passwords, request bodies, and arbitrary user input
// outside the logging contract.
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

	if event.UserID != "" {
		fields = append(fields, zap.String("user_id", event.UserID))
	}

	if event.Role != "" {
		fields = append(fields, zap.String("role", event.Role))
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
