package observability_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure/observability"
)

func TestNewMetrics(t *testing.T) {
	metrics := observability.NewMetrics()

	if metrics == nil {
		t.Fatal("expected metrics collector")
	}
}

func TestMetricsIncrement(t *testing.T) {
	metrics := observability.NewMetrics()

	metric := ports.Metric{
		Name: "payment_attempts_total",
		Labels: map[string]string{
			"operation": "create_payment",
			"result":    "success",
		},
	}

	for range 3 {
		if err := metrics.Increment(context.Background(), metric); err != nil {
			t.Fatalf("increment metric: %v", err)
		}
	}

	if got := metrics.CounterValue(metric.Name, metric.Labels); got != 3 {
		t.Fatalf("expected counter value 3, got %v", got)
	}
}

func TestMetricsObserve(t *testing.T) {
	metrics := observability.NewMetrics()

	metric := ports.Metric{
		Name:  "payment_operation_duration",
		Value: 1.25,
		Labels: map[string]string{
			"operation": "create_payment",
		},
	}

	if err := metrics.Observe(context.Background(), metric); err != nil {
		t.Fatalf("observe metric: %v", err)
	}

	values := metrics.ObservationValues(metric.Name, metric.Labels)

	if len(values) != 1 {
		t.Fatalf("expected one observation, got %d", len(values))
	}

	if values[0] != 1.25 {
		t.Fatalf("expected observation 1.25, got %v", values[0])
	}
}

func TestMetricsRejectEmptyName(t *testing.T) {
	metrics := observability.NewMetrics()

	err := metrics.Increment(
		context.Background(),
		ports.Metric{Name: "   "},
	)

	if !errors.Is(err, observability.ErrEmptyMetricName) {
		t.Fatalf("expected ErrEmptyMetricName, got %v", err)
	}

	err = metrics.Observe(
		context.Background(),
		ports.Metric{Name: ""},
	)

	if !errors.Is(err, observability.ErrEmptyMetricName) {
		t.Fatalf("expected ErrEmptyMetricName, got %v", err)
	}
}

func TestMetricsLabelsAreOrderIndependent(t *testing.T) {
	metrics := observability.NewMetrics()

	firstLabels := map[string]string{
		"operation": "create_payment",
		"result":    "success",
	}

	secondLabels := map[string]string{
		"result":    "success",
		"operation": "create_payment",
	}

	if err := metrics.Increment(
		context.Background(),
		ports.Metric{Name: "payment_attempts_total", Labels: firstLabels},
	); err != nil {
		t.Fatalf("increment metric: %v", err)
	}

	if got := metrics.CounterValue(
		"payment_attempts_total",
		secondLabels,
	); got != 1 {
		t.Fatalf("expected counter value 1, got %v", got)
	}
}

func TestMetricsObservationValuesReturnsCopy(t *testing.T) {
	metrics := observability.NewMetrics()

	metric := ports.Metric{
		Name:  "payment_duration",
		Value: 1,
	}

	if err := metrics.Observe(context.Background(), metric); err != nil {
		t.Fatalf("observe metric: %v", err)
	}

	values := metrics.ObservationValues(metric.Name)
	values[0] = 999

	recorded := metrics.ObservationValues(metric.Name)

	if recorded[0] != 1 {
		t.Fatalf("expected stored observation to remain 1, got %v", recorded[0])
	}
}

func TestMetricsIncrementConcurrent(t *testing.T) {
	metrics := observability.NewMetrics()

	const workers = 100

	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			if err := metrics.Increment(
				context.Background(),
				ports.Metric{Name: "payment_attempts_total"},
			); err != nil {
				t.Errorf("increment metric: %v", err)
			}
		}()
	}

	wg.Wait()

	if got := metrics.CounterValue("payment_attempts_total"); got != workers {
		t.Fatalf("expected counter value %d, got %v", workers, got)
	}
}

func TestNilMetricsReturnsErrors(t *testing.T) {
	var metrics *observability.Metrics

	if err := metrics.Increment(
		context.Background(),
		ports.Metric{Name: "payment_test"},
	); err == nil {
		t.Fatal("expected error from nil metrics")
	}

	if err := metrics.Observe(
		context.Background(),
		ports.Metric{Name: "payment_test"},
	); err == nil {
		t.Fatal("expected error from nil metrics")
	}

	if got := metrics.CounterValue("payment_test"); got != 0 {
		t.Fatalf("expected zero counter, got %v", got)
	}

	if got := metrics.ObservationValues("payment_test"); got != nil {
		t.Fatalf("expected nil observations, got %v", got)
	}
}
