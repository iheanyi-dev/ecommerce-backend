package ports_test

import (
	"context"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
)

func TestMetricsInterface_CanBeImplemented(t *testing.T) {
	var metrics ports.Metrics = &testMetrics{}

	err := metrics.Increment(
		context.Background(),
		ports.Metric{
			Name: "auth.login.attempts",
		},
	)
	if err != nil {
		t.Fatalf("expected metric increment to succeed, got %v", err)
	}

	err = metrics.Observe(
		context.Background(),
		ports.Metric{
			Name:  "http.request.duration",
			Value: 25,
		},
	)
	if err != nil {
		t.Fatalf("expected metric observation to succeed, got %v", err)
	}
}

// testMetrics is a minimal implementation used only to verify that the
// application-level Metrics contract can be implemented without depending
// on a concrete monitoring system.
type testMetrics struct{}

func (m *testMetrics) Increment(
	ctx context.Context,
	metric ports.Metric,
) error {
	return nil
}

func (m *testMetrics) Observe(
	ctx context.Context,
	metric ports.Metric,
) error {
	return nil
}
