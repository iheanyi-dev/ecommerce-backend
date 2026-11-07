package observability

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"sync"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/ports"
)

// StructuredLogger writes application events as JSON.
type StructuredLogger struct {
	output io.Writer
	mu     sync.Mutex
}

// NewStructuredLogger creates a structured logger writing to the supplied
// output. When output is nil, standard output is used.
func NewStructuredLogger(output io.Writer) *StructuredLogger {
	if output == nil {
		output = os.Stdout
	}

	return &StructuredLogger{
		output: output,
	}
}

// Log writes one structured application event.
func (l *StructuredLogger) Log(
	_ context.Context,
	event ports.LogEvent,
) error {
	payload := map[string]string{
		"event": event.Event,
	}

	if event.Operation != "" {
		payload["operation"] = event.Operation
	}

	if event.FailureCategory != "" {
		payload["failure_category"] = event.FailureCategory
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	_, err = l.output.Write(append(encoded, '\n'))
	return err
}

var _ ports.Logger = (*StructuredLogger)(nil)
