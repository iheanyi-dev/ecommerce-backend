package observability

import (
	"context"
	"errors"
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
)

var (
	ErrNilWriter = errors.New("logger writer cannot be nil")
)

// Logger implements the Payment application logger port.
type Logger struct {
	logger *zap.Logger
}

var _ ports.Logger = (*Logger)(nil)

// NewLogger creates a structured JSON logger writing to the supplied
// destination.
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

// NewProductionLogger creates a logger that writes to stdout.
func NewProductionLogger() (*Logger, error) {
	return NewLogger(os.Stdout)
}

// Log records a safe structured Payment application event.
//
// Only fields explicitly defined by ports.LogEvent are emitted.
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
