package observability_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/infrastructure/observability"
)

func TestNewLoggerRejectsNilWriter(t *testing.T) {
	logger, err := observability.NewLogger(nil)
	if !errors.Is(err, observability.ErrNilWriter) {
		t.Fatalf("NewLogger(nil) error = %v, want %v", err, observability.ErrNilWriter)
	}

	if logger != nil {
		t.Fatal("expected nil logger when writer is nil")
	}
}

func TestLoggerWritesStructuredWhitelistedFields(t *testing.T) {
	var output bytes.Buffer

	logger, err := observability.NewLogger(&output)
	if err != nil {
		t.Fatalf("NewLogger() returned an unexpected error: %v", err)
	}

	event := ports.LogEvent{
		Event:           "order_created",
		Operation:       "create_order",
		UserID:          "customer-123",
		Role:            "customer",
		FailureCategory: "",
	}

	if err := logger.Log(context.Background(), event); err != nil {
		t.Fatalf("Log() returned an unexpected error: %v", err)
	}

	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("log output is not valid JSON: %v", err)
	}

	expected := map[string]string{
		"msg":              "order_application_event",
		"event":            "order_created",
		"operation":        "create_order",
		"user_id":          "customer-123",
		"role":             "customer",
		"failure_category": "",
	}

	for key, want := range expected {
		got, ok := record[key]
		if !ok {
			t.Errorf("log record is missing field %q", key)
			continue
		}

		if got != want {
			t.Errorf("log field %q = %v, want %q", key, got, want)
		}
	}

	if strings.Contains(output.String(), "password") {
		t.Fatal("log unexpectedly contains a credential-like field")
	}
}

func TestNilLoggerReturnsError(t *testing.T) {
	var logger *observability.Logger

	err := logger.Log(context.Background(), ports.LogEvent{})
	if !errors.Is(err, observability.ErrNilLogger) {
		t.Fatalf("Log() error = %v, want %v", err, observability.ErrNilLogger)
	}
}
