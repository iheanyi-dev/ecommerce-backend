package observability_test

import (
	"reflect"
	"sync"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/infrastructure/observability"
)

func TestMetricsRecordAndRetrieveValues(t *testing.T) {
	metrics := observability.NewMetrics()
	labels := map[string]string{"operation": "save", "result": "success"}

	metrics.Record(ports.Metric{Name: "cart.operations", Value: 1, Labels: labels})
	metrics.Record(ports.Metric{Name: "cart.operations", Value: 2, Labels: labels})

	got := metrics.ObservationValues("cart.operations", labels)
	want := []float64{1, 2}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("values = %v, want %v", got, want)
	}
}

func TestMetricsSeparateLabelSets(t *testing.T) {
	metrics := observability.NewMetrics()

	metrics.Record(ports.Metric{
		Name: "cart.operations", Value: 1,
		Labels: map[string]string{"operation": "save", "result": "success"},
	})
	metrics.Record(ports.Metric{
		Name: "cart.operations", Value: 3,
		Labels: map[string]string{"result": "success", "operation": "save"},
	})
	metrics.Record(ports.Metric{
		Name: "cart.operations", Value: 4,
		Labels: map[string]string{"operation": "delete", "result": "success"},
	})

	got := metrics.ObservationValues(
		"cart.operations",
		map[string]string{"operation": "save", "result": "success"},
	)
	if !reflect.DeepEqual(got, []float64{1, 3}) {
		t.Fatalf("values = %v, want [1 3]", got)
	}

	other := metrics.ObservationValues(
		"cart.operations",
		map[string]string{"operation": "delete", "result": "success"},
	)
	if !reflect.DeepEqual(other, []float64{4}) {
		t.Fatalf("other label values = %v, want [4]", other)
	}
}

func TestMetricsReturnCopyOfObservations(t *testing.T) {
	metrics := observability.NewMetrics()
	metrics.Record(ports.Metric{Name: "cart.operations", Value: 1})

	values := metrics.ObservationValues("cart.operations")
	values[0] = 99

	got := metrics.ObservationValues("cart.operations")
	if !reflect.DeepEqual(got, []float64{1}) {
		t.Fatalf("stored values were modified: %v", got)
	}
}

func TestMetricsIgnoreEmptyName(t *testing.T) {
	metrics := observability.NewMetrics()
	metrics.Record(ports.Metric{Name: "  ", Value: 10})

	if got := metrics.ObservationValues("  "); len(got) != 0 {
		t.Fatalf("expected no observations, got %v", got)
	}
}

func TestMetricsConcurrentRecording(t *testing.T) {
	metrics := observability.NewMetrics()

	const workers = 20
	const recordsPerWorker = 50

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range recordsPerWorker {
				metrics.Record(ports.Metric{Name: "cart.operations", Value: 1})
			}
		}()
	}
	wg.Wait()

	got := metrics.ObservationValues("cart.operations")
	if len(got) != workers*recordsPerWorker {
		t.Fatalf("record count = %d, want %d",
			len(got), workers*recordsPerWorker)
	}
}
