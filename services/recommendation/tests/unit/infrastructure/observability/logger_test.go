package observability_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/infrastructure/observability"
)

type failingWriter struct{}

func (w failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestStructuredLoggerWritesJSONEvent(t *testing.T) {
	var output bytes.Buffer
	logger := observability.NewStructuredLogger(&output)

	err := logger.Log(context.Background(), ports.LogEvent{
		Event:           "recommendation.completed",
		Operation:       "find_similar_products",
		FailureCategory: "none",
	})
	if err != nil {
		t.Fatalf("log event: %v", err)
	}

	var payload map[string]string
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &payload); err != nil {
		t.Fatalf("decode JSON output: %v", err)
	}

	expected := map[string]string{
		"event":            "recommendation.completed",
		"operation":        "find_similar_products",
		"failure_category": "none",
	}

	for key, value := range expected {
		if payload[key] != value {
			t.Errorf("expected %s=%q, got %q", key, value, payload[key])
		}
	}
}

func TestStructuredLoggerOmitsEmptyOptionalFields(t *testing.T) {
	var output bytes.Buffer
	logger := observability.NewStructuredLogger(&output)

	if err := logger.Log(context.Background(), ports.LogEvent{
		Event: "recommendation.started",
	}); err != nil {
		t.Fatalf("log event: %v", err)
	}

	var payload map[string]string
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &payload); err != nil {
		t.Fatalf("decode JSON output: %v", err)
	}

	if payload["event"] != "recommendation.started" {
		t.Fatalf("unexpected event: %q", payload["event"])
	}

	if _, exists := payload["operation"]; exists {
		t.Error("expected operation field to be omitted")
	}
	if _, exists := payload["failure_category"]; exists {
		t.Error("expected failure_category field to be omitted")
	}
}

func TestStructuredLoggerReturnsWriterError(t *testing.T) {
	logger := observability.NewStructuredLogger(failingWriter{})

	err := logger.Log(context.Background(), ports.LogEvent{
		Event: "recommendation.failed",
	})
	if err == nil {
		t.Fatal("expected writer error")
	}
	if !strings.Contains(err.Error(), "write failed") {
		t.Fatalf("expected writer error, got %v", err)
	}
}

func TestStructuredLoggerSupportsConcurrentWrites(t *testing.T) {
	var output bytes.Buffer
	logger := observability.NewStructuredLogger(&output)

	const workers = 20

	var wg sync.WaitGroup
	errorsFound := make(chan error, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)

		go func(index int) {
			defer wg.Done()

			err := logger.Log(context.Background(), ports.LogEvent{
				Event:     fmt.Sprintf("recommendation.event.%d", index),
				Operation: "concurrent_test",
			})
			if err != nil {
				errorsFound <- err
			}
		}(i)
	}

	wg.Wait()
	close(errorsFound)

	for err := range errorsFound {
		t.Errorf("concurrent log: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != workers {
		t.Fatalf("expected %d log entries, got %d", workers, len(lines))
	}

	for i, line := range lines {
		var payload map[string]string
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			t.Errorf("entry %d contains invalid JSON: %v", i, err)
		}
	}
}
