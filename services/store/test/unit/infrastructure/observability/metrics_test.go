package observability_test

import (
	"context"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/observability"
)

func TestMetrics_IncrementAndObserve(t *testing.T) {
	metrics := observability.NewMetrics()
	ctx := context.Background()

	labels := map[string]string{
		"operation": "create_store",
		"result":    "success",
	}

	if err := metrics.Increment(
		ctx,
		ports.Metric{
			Name:   "store.operations.total",
			Value:  1,
			Labels: labels,
		},
	); err != nil {
		t.Fatalf("increment metric: %v", err)
	}

	if got := metrics.CounterValue(
		"store.operations.total",
		labels,
	); got != 1 {
		t.Fatalf("counter value: got %v, want 1", got)
	}

	if err := metrics.Observe(
		ctx,
		ports.Metric{
			Name:   "store.operation.duration_ms",
			Value:  12.5,
			Labels: labels,
		},
	); err != nil {
		t.Fatalf("observe metric: %v", err)
	}

	values := metrics.ObservationValues(
		"store.operation.duration_ms",
		labels,
	)

	if len(values) != 1 {
		t.Fatalf("observation count: got %d, want 1", len(values))
	}

	if values[0] != 12.5 {
		t.Fatalf("observation value: got %v, want 12.5", values[0])
	}
}

func TestMetrics_EquivalentLabelMapsHaveSameIdentity(t *testing.T) {
	metrics := observability.NewMetrics()
	ctx := context.Background()

	firstLabels := map[string]string{
		"operation": "create_store",
		"result":    "success",
	}

	secondLabels := map[string]string{
		"result":    "success",
		"operation": "create_store",
	}

	if err := metrics.Increment(
		ctx,
		ports.Metric{
			Name:   "store.operations.total",
			Labels: firstLabels,
		},
	); err != nil {
		t.Fatalf("increment metric: %v", err)
	}

	if got := metrics.CounterValue(
		"store.operations.total",
		secondLabels,
	); got != 1 {
		t.Fatalf(
			"equivalent label maps produced different metric: got %v, want 1",
			got,
		)
	}
}

func TestMetrics_RejectsEmptyName(t *testing.T) {
	metrics := observability.NewMetrics()

	err := metrics.Increment(
		context.Background(),
		ports.Metric{},
	)

	if err != observability.ErrEmptyMetricName {
		t.Fatalf(
			"expected ErrEmptyMetricName, got %v",
			err,
		)
	}
}
