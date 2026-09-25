package observability_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/observability"
)

func TestLogger_LogWritesSafeStructuredEvent(t *testing.T) {
	var output bytes.Buffer

	logger, err := observability.NewLogger(&output)
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	err = logger.Log(
		context.Background(),
		ports.LogEvent{
			Event:           "store.create.succeeded",
			Operation:       "create_store",
			UserID:          "00000000-0000-0000-0000-000000000001",
			Role:            "vendor",
			FailureCategory: "",
		},
	)
	if err != nil {
		t.Fatalf("log event: %v", err)
	}

	var record map[string]any

	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("decode structured log: %v", err)
	}

	if record["event"] != "store.create.succeeded" {
		t.Fatalf("event mismatch: got %v", record["event"])
	}

	if record["operation"] != "create_store" {
		t.Fatalf("operation mismatch: got %v", record["operation"])
	}

	if record["user_id"] != "00000000-0000-0000-0000-000000000001" {
		t.Fatalf("user_id mismatch: got %v", record["user_id"])
	}

	if record["role"] != "vendor" {
		t.Fatalf("role mismatch: got %v", record["role"])
	}
}

func TestLogger_RejectsNilWriter(t *testing.T) {
	_, err := observability.NewLogger(nil)

	if err != observability.ErrNilWriter {
		t.Fatalf("expected ErrNilWriter, got %v", err)
	}
}
