package observability_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/infrastructure/observability"
)

func TestNewLoggerRejectsNilWriter(t *testing.T) {
	_, err := observability.NewLogger(nil)
	if !errors.Is(err, observability.ErrNilWriter) {
		t.Fatalf("expected ErrNilWriter, got %v", err)
	}
}

func TestLoggerWritesStructuredEvent(t *testing.T) {
	var output bytes.Buffer

	logger, err := observability.NewLogger(&output)
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	logger.LogEvent(ports.LogEvent{
		Event:           "cart.save.failed",
		Operation:       "save_cart",
		FailureCategory: "persistence",
	})

	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("decode JSON log: %v; output: %s", err, output.String())
	}

	for key, want := range map[string]string{
		"event":            "cart.save.failed",
		"msg":              "cart.save.failed",
		"operation":        "save_cart",
		"failure_category": "persistence",
	} {
		if got, ok := record[key].(string); !ok || got != want {
			t.Errorf("%s = %v, want %q", key, record[key], want)
		}
	}
}

func TestLoggerOmitsEmptyFailureCategory(t *testing.T) {
	var output bytes.Buffer

	logger, err := observability.NewLogger(&output)
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	logger.LogEvent(ports.LogEvent{
		Event:     "cart.read",
		Operation: "get_cart",
	})

	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("decode JSON log: %v", err)
	}

	if _, exists := record["failure_category"]; exists {
		t.Fatal("empty failure_category should be omitted")
	}
}
