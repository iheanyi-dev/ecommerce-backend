package observability_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure/observability"
)

func TestNewLoggerRejectsNilWriter(t *testing.T) {
	_, err := observability.NewLogger(nil)

	if !errors.Is(err, observability.ErrNilWriter) {
		t.Fatalf("expected ErrNilWriter, got %v", err)
	}
}

func TestLoggerLogWritesStructuredEvent(t *testing.T) {
	var output bytes.Buffer

	logger, err := observability.NewLogger(&output)

	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	event := ports.LogEvent{
		Event:           "payment.create.succeeded",
		Operation:       "create_payment",
		UserID:          "user-123",
		Role:            "customer",
		FailureCategory: "none",
	}

	if err := logger.Log(context.Background(), event); err != nil {
		t.Fatalf("log event: %v", err)
	}

	var recorded map[string]interface{}

	if err := json.Unmarshal(output.Bytes(), &recorded); err != nil {
		t.Fatalf("decode log output: %v", err)
	}

	expected := map[string]string{
		"event":            event.Event,
		"operation":        event.Operation,
		"user_id":          event.UserID,
		"role":             event.Role,
		"failure_category": event.FailureCategory,
	}

	for key, value := range expected {
		if recorded[key] != value {
			t.Errorf("expected %s=%q, got %v", key, value, recorded[key])
		}
	}

	if _, exists := recorded["timestamp"]; !exists {
		t.Error("expected timestamp field")
	}
}

func TestLoggerOmitsOptionalFields(t *testing.T) {
	var output bytes.Buffer

	logger, err := observability.NewLogger(&output)

	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	if err := logger.Log(
		context.Background(),
		ports.LogEvent{
			Event:     "payment.test",
			Operation: "test",
		},
	); err != nil {
		t.Fatalf("log event: %v", err)
	}

	var recorded map[string]interface{}

	if err := json.Unmarshal(output.Bytes(), &recorded); err != nil {
		t.Fatalf("decode log output: %v", err)
	}

	for _, key := range []string{
		"user_id",
		"role",
		"failure_category",
	} {
		if _, exists := recorded[key]; exists {
			t.Errorf("did not expect optional field %q", key)
		}
	}
}

func TestNilLoggerReturnsError(t *testing.T) {
	var logger *observability.Logger

	err := logger.Log(
		context.Background(),
		ports.LogEvent{Event: "payment.test"},
	)

	if err == nil {
		t.Fatal("expected error from nil logger")
	}
}

func TestNewProductionLogger(t *testing.T) {
	logger, err := observability.NewProductionLogger()

	if err != nil {
		t.Fatalf("create production logger: %v", err)
	}

	if logger == nil {
		t.Fatal("expected production logger")
	}
}
