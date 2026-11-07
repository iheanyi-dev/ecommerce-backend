package observability_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/infrastructure/observability"
)

func TestMetricsIncrement(t *testing.T) {
	metrics := observability.NewMetrics()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if err := metrics.Increment(ctx, ports.Metric{Name: "recommendations.total"}); err != nil {
			t.Fatalf("increment metric: %v", err)
		}
	}

	if got := metrics.CounterValue("recommendations.total"); got != 3 {
		t.Fatalf("expected counter value 3, got %f", got)
	}
}

func TestMetricsObserve(t *testing.T) {
	metrics := observability.NewMetrics()
	ctx := context.Background()

	values := []float64{0.12, 0.25, 0.37}
	for _, value := range values {
		if err := metrics.Observe(ctx, ports.Metric{
			Name:  "recommendations.duration",
			Value: value,
		}); err != nil {
			t.Fatalf("observe metric: %v", err)
		}
	}

	got := metrics.ObservationValues("recommendations.duration")
	if len(got) != len(values) {
		t.Fatalf("expected %d observations, got %d", len(values), len(got))
	}

	for i, expected := range values {
		if got[i] != expected {
			t.Errorf("observation %d: expected %f, got %f", i, expected, got[i])
		}
	}
}

func TestMetricsSeparatesLabels(t *testing.T) {
	metrics := observability.NewMetrics()
	ctx := context.Background()

	first := ports.Metric{
		Name:   "recommendations.total",
		Labels: map[string]string{"operation": "products"},
	}
	second := ports.Metric{
		Name:   "recommendations.total",
		Labels: map[string]string{"operation": "stores"},
	}

	if err := metrics.Increment(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := metrics.Increment(ctx, second); err != nil {
		t.Fatal(err)
	}
	if err := metrics.Increment(ctx, first); err != nil {
		t.Fatal(err)
	}

	if got := metrics.CounterValue(first.Name, first.Labels); got != 2 {
		t.Fatalf("expected product counter 2, got %f", got)
	}
	if got := metrics.CounterValue(second.Name, second.Labels); got != 1 {
		t.Fatalf("expected store counter 1, got %f", got)
	}
}

func TestMetricsRejectsEmptyMetricName(t *testing.T) {
	metrics := observability.NewMetrics()
	ctx := context.Background()

	if err := metrics.Increment(ctx, ports.Metric{Name: " "}); !errors.Is(err, observability.ErrEmptyMetricName) {
		t.Fatalf("expected empty metric name error, got %v", err)
	}
	if err := metrics.Observe(ctx, ports.Metric{Name: ""}); !errors.Is(err, observability.ErrEmptyMetricName) {
		t.Fatalf("expected empty metric name error, got %v", err)
	}
}

func TestMetricsReturnsCopyOfObservations(t *testing.T) {
	metrics := observability.NewMetrics()
	ctx := context.Background()

	if err := metrics.Observe(ctx, ports.Metric{Name: "duration", Value: 1}); err != nil {
		t.Fatal(err)
	}

	values := metrics.ObservationValues("duration")
	values[0] = 99

	if got := metrics.ObservationValues("duration")[0]; got != 1 {
		t.Fatalf("expected stored observation to remain 1, got %f", got)
	}
}

func TestMetricsSupportsConcurrentUpdates(t *testing.T) {
	metrics := observability.NewMetrics()
	ctx := context.Background()

	const workers = 50
	const iterations = 100

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				if err := metrics.Increment(ctx, ports.Metric{Name: "concurrent.counter"}); err != nil {
					t.Errorf("increment metric: %v", err)
				}
				if err := metrics.Observe(ctx, ports.Metric{Name: "concurrent.observation", Value: 1}); err != nil {
					t.Errorf("observe metric: %v", err)
				}
			}
		}()
	}

	wg.Wait()

	expected := float64(workers * iterations)
	if got := metrics.CounterValue("concurrent.counter"); got != expected {
		t.Fatalf("expected counter %f, got %f", expected, got)
	}
	if got := len(metrics.ObservationValues("concurrent.observation")); got != int(expected) {
		t.Fatalf("expected %d observations, got %d", int(expected), got)
	}
}

func TestNilMetricsReturnsErrorsOrZeroValues(t *testing.T) {
	var metrics *observability.Metrics

	if err := metrics.Increment(context.Background(), ports.Metric{Name: "counter"}); err == nil {
		t.Fatal("expected error from nil metrics")
	}
	if err := metrics.Observe(context.Background(), ports.Metric{Name: "observation"}); err == nil {
		t.Fatal("expected error from nil metrics")
	}
	if got := metrics.CounterValue("counter"); got != 0 {
		t.Fatalf("expected zero counter, got %f", got)
	}
	if got := metrics.ObservationValues("observation"); got != nil {
		t.Fatalf("expected nil observations, got %v", got)
	}
}
