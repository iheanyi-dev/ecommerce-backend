package observability

import (
	"errors"
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/ports"
)

var ErrNilWriter = errors.New("logger writer cannot be nil")

// Logger implements the Cart application logging contract.
type Logger struct {
	logger *zap.Logger
}

var _ ports.Logger = (*Logger)(nil)

// NewLogger creates a structured JSON logger writing to the supplied destination.
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

	return &Logger{logger: zap.New(core)}, nil
}

// NewProductionLogger creates a logger that writes structured JSON to stdout.
func NewProductionLogger() (*Logger, error) {
	return NewLogger(os.Stdout)
}

// LogEvent emits only fields explicitly permitted by the Cart logging contract.
func (l *Logger) LogEvent(event ports.LogEvent) {
	if l == nil || l.logger == nil {
		return
	}

	fields := []zap.Field{
		zap.String("event", event.Event),
		zap.String("operation", event.Operation),
	}

	if event.FailureCategory != "" {
		fields = append(fields,
			zap.String("failure_category", event.FailureCategory),
		)
	}

	l.logger.Info(event.Event, fields...)
}
