package observability_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/observability"
)

func TestLogger_Log_WritesStructuredJSON(t *testing.T) {
	var output bytes.Buffer

	logger, err := observability.NewLogger(&output)
	if err != nil {
		t.Fatalf("expected logger creation to succeed, got %v", err)
	}

	err = logger.Log(
		context.Background(),
		ports.LogEvent{
			Event:          "auth.login.succeeded",
			Operation:      "login",
			UserID:         "user-123",
			Role:           "vendor",
			HTTPMethod:     "POST",
			Route:          "/api/v1/auth/login",
			StatusCode:     200,
			DurationMillis: 15,
		},
	)
	if err != nil {
		t.Fatalf("expected logging to succeed, got %v", err)
	}

	var logged map[string]interface{}

	if err := json.Unmarshal(output.Bytes(), &logged); err != nil {
		t.Fatalf(
			"expected logger to write valid JSON, got %v; output: %s",
			err,
			output.String(),
		)
	}

	if _, exists := logged["timestamp"]; !exists {
		t.Fatalf(
			"expected JSON field %q to exist; log entry: %#v",
			"timestamp",
			logged,
		)
	}

	assertJSONField(t, logged, "event", "auth.login.succeeded")
	assertJSONField(t, logged, "operation", "login")
	assertJSONField(t, logged, "user_id", "user-123")
	assertJSONField(t, logged, "role", "vendor")
	assertJSONField(t, logged, "http_method", "POST")
	assertJSONField(t, logged, "route", "/api/v1/auth/login")
	assertJSONField(t, logged, "status_code", float64(200))
	assertJSONField(t, logged, "duration_ms", float64(15))
}

func TestLogger_Log_DoesNotLogSensitiveAuthenticationData(t *testing.T) {
	var output bytes.Buffer

	logger, err := observability.NewLogger(&output)
	if err != nil {
		t.Fatalf("expected logger creation to succeed, got %v", err)
	}

	// These values intentionally represent secrets that must never be
	// accepted as logging fields. The LogEvent contract does not provide
	// fields for them, which is part of the security boundary.
	err = logger.Log(
		context.Background(),
		ports.LogEvent{
			Event:     "auth.login.succeeded",
			Operation: "login",
			UserID:    "user-123",
			Role:      "user",
		},
	)
	if err != nil {
		t.Fatalf("expected logging to succeed, got %v", err)
	}

	loggedOutput := output.String()

	for _, sensitiveValue := range []string{
		"password",
		"password_hash",
		"access_token",
		"refresh_token",
		"Authorization",
		"Bearer",
	} {
		if bytes.Contains(
			[]byte(loggedOutput),
			[]byte(sensitiveValue),
		) {
			t.Fatalf(
				"expected sensitive value %q not to appear in logs; output: %s",
				sensitiveValue,
				loggedOutput,
			)
		}
	}
}

func TestNewLogger_RejectsNilWriter(t *testing.T) {
	_, err := observability.NewLogger(nil)

	if err == nil {
		t.Fatal("expected nil writer to be rejected")
	}
}

// assertJSONField verifies a structured JSON field without coupling the test
// to Zap's internal representation.
func assertJSONField(
	t *testing.T,
	logged map[string]interface{},
	field string,
	expected interface{},
) {
	t.Helper()

	actual, exists := logged[field]
	if !exists {
		t.Fatalf(
			"expected JSON field %q to exist; log entry: %#v",
			field,
			logged,
		)
	}

	if actual != expected {
		t.Fatalf(
			"expected field %q to equal %#v, got %#v",
			field,
			expected,
			actual,
		)
	}
}
func TestNewProductionLogger_UsesStandardOutput(t *testing.T) {
	logger, err := observability.NewProductionLogger()
	if err != nil {
		t.Fatalf(
			"expected production logger creation to succeed, got %v",
			err,
		)
	}

	if logger == nil {
		t.Fatal("expected production logger not to be nil")
	}
}
func TestNewProductionLogger_ImplementsApplicationLogger(
	t *testing.T,
) {
	var logger ports.Logger

	concreteLogger, err := observability.NewProductionLogger()
	if err != nil {
		t.Fatalf(
			"expected production logger creation to succeed, got %v",
			err,
		)
	}

	logger = concreteLogger

	if logger == nil {
		t.Fatal("expected logger to implement ports.Logger")
	}
}
