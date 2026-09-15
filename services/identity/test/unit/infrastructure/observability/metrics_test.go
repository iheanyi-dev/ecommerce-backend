package observability_test

import (
	"context"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/observability"
)

func TestMetrics_IncrementRecordsCounter(t *testing.T) {
	metrics := observability.NewMetrics()

	err := metrics.Increment(
		context.Background(),
		ports.Metric{
			Name: "auth.login.attempts",
		},
	)
	if err != nil {
		t.Fatalf("expected metric increment to succeed, got %v", err)
	}

	value := metrics.CounterValue("auth.login.attempts")
	if value != 1 {
		t.Fatalf(
			"expected counter value 1, got %v",
			value,
		)
	}
}

func TestMetrics_IncrementAccumulatesCounterValues(t *testing.T) {
	metrics := observability.NewMetrics()

	metric := ports.Metric{
		Name: "auth.login.failures",
	}

	if err := metrics.Increment(context.Background(), metric); err != nil {
		t.Fatalf("expected first increment to succeed, got %v", err)
	}

	if err := metrics.Increment(context.Background(), metric); err != nil {
		t.Fatalf("expected second increment to succeed, got %v", err)
	}

	value := metrics.CounterValue("auth.login.failures")
	if value != 2 {
		t.Fatalf(
			"expected counter value 2, got %v",
			value,
		)
	}
}

func TestMetrics_ObserveRecordsMeasurement(t *testing.T) {
	metrics := observability.NewMetrics()

	err := metrics.Observe(
		context.Background(),
		ports.Metric{
			Name:  "http.request.duration",
			Value: 25,
		},
	)
	if err != nil {
		t.Fatalf("expected metric observation to succeed, got %v", err)
	}

	values := metrics.ObservationValues("http.request.duration")

	if len(values) != 1 {
		t.Fatalf(
			"expected 1 observation, got %d",
			len(values),
		)
	}

	if values[0] != 25 {
		t.Fatalf(
			"expected observation value 25, got %v",
			values[0],
		)
	}
}

func TestMetrics_PreservesUsefulLowCardinalityLabels(t *testing.T) {
	metrics := observability.NewMetrics()

	err := metrics.Increment(
		context.Background(),
		ports.Metric{
			Name: "http.requests",
			Labels: map[string]string{
				"method": "GET",
				"route":  "/api/v1/users/me",
				"status": "200",
			},
		},
	)
	if err != nil {
		t.Fatalf("expected metric increment to succeed, got %v", err)
	}

	value := metrics.CounterValue(
		"http.requests",
		map[string]string{
			"method": "GET",
			"route":  "/api/v1/users/me",
			"status": "200",
		},
	)

	if value != 1 {
		t.Fatalf(
			"expected labelled counter value 1, got %v",
			value,
		)
	}
}
