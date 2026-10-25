package observability_test

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/infrastructure/observability"
)

func TestMetricsIncrementUsesNameAndLabels(t *testing.T) {
	metrics := observability.NewMetrics()
	ctx := context.Background()

	first := ports.Metric{
		Name:  "orders_created",
		Value: 2,
		Labels: map[string]string{
			"store": "store-a",
			"role":  "customer",
		},
	}
	second := ports.Metric{
		Name:  "orders_created",
		Value: 3,
		Labels: map[string]string{
			"role":  "customer",
			"store": "store-a",
		},
	}

	if err := metrics.Increment(ctx, first); err != nil {
		t.Fatalf("first Increment() returned an unexpected error: %v", err)
	}
	if err := metrics.Increment(ctx, second); err != nil {
		t.Fatalf("second Increment() returned an unexpected error: %v", err)
	}

	if got := metrics.CounterValue(first.Name, first.Labels); got != 5 {
		t.Errorf("CounterValue() = %v, want 5", got)
	}

	if got := metrics.CounterValue(first.Name, map[string]string{"store": "store-b", "role": "customer"}); got != 0 {
		t.Errorf("CounterValue() for different labels = %v, want 0", got)
	}
}

func TestMetricsObserveReturnsCopyOfValues(t *testing.T) {
	metrics := observability.NewMetrics()
	ctx := context.Background()

	metric := ports.Metric{
		Name:   "order_duration_seconds",
		Value:  0.25,
		Labels: map[string]string{"operation": "create"},
	}

	if err := metrics.Observe(ctx, metric); err != nil {
		t.Fatalf("first Observe() returned an unexpected error: %v", err)
	}

	metric.Value = 0.75
	if err := metrics.Observe(ctx, metric); err != nil {
		t.Fatalf("second Observe() returned an unexpected error: %v", err)
	}

	values := metrics.ObservationValues(metric.Name, metric.Labels)
	if !reflect.DeepEqual(values, []float64{0.25, 0.75}) {
		t.Fatalf("ObservationValues() = %v, want [0.25 0.75]", values)
	}

	values[0] = 99
	again := metrics.ObservationValues(metric.Name, metric.Labels)
	if !reflect.DeepEqual(again, []float64{0.25, 0.75}) {
		t.Fatalf("mutating returned observations changed internal state: %v", again)
	}
}

func TestMetricsRejectEmptyName(t *testing.T) {
	metrics := observability.NewMetrics()

	if err := metrics.Increment(context.Background(), ports.Metric{}); !errors.Is(err, observability.ErrEmptyMetricName) {
		t.Errorf("Increment() error = %v, want %v", err, observability.ErrEmptyMetricName)
	}

	if err := metrics.Observe(context.Background(), ports.Metric{}); !errors.Is(err, observability.ErrEmptyMetricName) {
		t.Errorf("Observe() error = %v, want %v", err, observability.ErrEmptyMetricName)
	}
}

func TestMetricsConcurrentIncrements(t *testing.T) {
	metrics := observability.NewMetrics()
	ctx := context.Background()

	const workers = 20
	const incrementsPerWorker = 100

	var waitGroup sync.WaitGroup
	for range workers {
		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			for range incrementsPerWorker {
				if err := metrics.Increment(ctx, ports.Metric{
					Name:  "concurrent_counter",
					Value: 1,
				}); err != nil {
					t.Errorf("Increment() returned an unexpected error: %v", err)
					return
				}
			}
		}()
	}

	waitGroup.Wait()

	want := float64(workers * incrementsPerWorker)
	if got := metrics.CounterValue("concurrent_counter", nil); got != want {
		t.Errorf("CounterValue() = %v, want %v", got, want)
	}
}
