package observability

import (
	"context"
	"errors"
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
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
// concrete Zap implementation and therefore keeps the logging library
// isolated inside Infrastructure.
type Logger struct {
	logger *zap.Logger
}

// NewLogger creates a structured JSON logger that writes to the supplied
// destination.
//
// A writer is supplied instead of hard-coding os.Stdout so that:
//   - production can write logs to standard output;
//   - tests can capture log output in memory;
//   - the logger remains independent of a particular runtime environment.
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

// NewProductionLogger creates the application's production logger.
//
// Production logs are written to standard output so that container runtimes,
// process supervisors, and centralized logging systems can collect them
// without the Identity service managing log files itself.
func NewProductionLogger() (*Logger, error) {
	return NewLogger(os.Stdout)
}

// Log records a structured application event.
//
// Only fields explicitly defined by ports.LogEvent are emitted. This is
// intentional: authentication credentials, tokens, passwords, password
// hashes, Authorization headers, and other secrets have no place in the
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

	if event.UserID != "" {
		fields = append(fields, zap.String("user_id", event.UserID))
	}

	if event.Role != "" {
		fields = append(fields, zap.String("role", event.Role))
	}

	if event.HTTPMethod != "" {
		fields = append(fields, zap.String("http_method", event.HTTPMethod))
	}

	if event.Route != "" {
		fields = append(fields, zap.String("route", event.Route))
	}

	if event.StatusCode != 0 {
		fields = append(fields, zap.Int("status_code", event.StatusCode))
	}

	if event.DurationMillis != 0 {
		fields = append(
			fields,
			zap.Int64("duration_ms", event.DurationMillis),
		)
	}

	if event.FailureCategory != "" {
		fields = append(
			fields,
			zap.String("failure_category", event.FailureCategory),
		)
	}

	if event.RequiredRole != "" {
		fields = append(
			fields,
			zap.String("required_role", event.RequiredRole),
		)
	}

	l.logger.Info(event.Event, fields...)

	return nil
}
