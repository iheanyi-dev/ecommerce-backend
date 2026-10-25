package observability

import (
	"context"
	"errors"
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
)

var (
	// ErrNilWriter indicates that the logger was created without a valid
	// destination for its structured log output.
	ErrNilWriter = errors.New("logger writer cannot be nil")

	// ErrNilLogger indicates that the logger has no underlying Zap logger.
	ErrNilLogger = errors.New("logger cannot be nil")
)

// Logger is the infrastructure implementation of the Order Logger port.
//
// It emits structured JSON logs and deliberately accepts only the fields
// defined by ports.LogEvent. Arbitrary request payloads and credentials are
// never added to log entries.
type Logger struct {
	logger *zap.Logger
}

// NewLogger creates a JSON logger that writes to the supplied destination.
//
// This constructor allows tests to capture log output without redirecting
// process-wide stdout or stderr.
func NewLogger(writer io.Writer) (*Logger, error) {
	if writer == nil {
		return nil, ErrNilWriter
	}

	encoderConfig := zap.NewProductionEncoderConfig()

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(writer),
		zapcore.InfoLevel,
	)

	return &Logger{
		logger: zap.New(core),
	}, nil
}

// NewProductionLogger creates the Order logger for normal service operation.
//
// Production logs are written to stdout so Docker and the deployment
// platform can collect and process them.
func NewProductionLogger() (*Logger, error) {
	return NewLogger(os.Stdout)
}

// Log emits one structured application event.
//
// Only explicitly approved fields from ports.LogEvent are written. The
// context is accepted to satisfy the application port; request identity and
// other fields are supplied explicitly by the caller.
func (l *Logger) Log(_ context.Context, event ports.LogEvent) error {
	if l == nil || l.logger == nil {
		return ErrNilLogger
	}

	fields := []zap.Field{
		zap.String("event", event.Event),
		zap.String("operation", event.Operation),
		zap.String("user_id", event.UserID),
		zap.String("role", event.Role),
		zap.String("failure_category", event.FailureCategory),
	}

	l.logger.Info("order_application_event", fields...)

	return nil
}

// Ensure Logger implements the application-layer logging contract.
var _ ports.Logger = (*Logger)(nil)
