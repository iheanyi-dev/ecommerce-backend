package ports_test

import (
	"context"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
)

func TestLoggerInterface_CanBeImplemented(t *testing.T) {
	var logger ports.Logger = &testLogger{}

	err := logger.Log(
		context.Background(),
		ports.LogEvent{
			Event:     "test.event",
			Operation: "test",
		},
	)
	if err != nil {
		t.Fatalf("expected logging to succeed, got %v", err)
	}
}

// testLogger is a minimal implementation used only to verify that the
// application-level Logger contract is correctly defined.
//
// The test deliberately does not depend on Zap or any other logging
// implementation.
type testLogger struct{}

func (l *testLogger) Log(
	ctx context.Context,
	event ports.LogEvent,
) error {
	return nil
}
