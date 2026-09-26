package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/presentation/middleware"
)

func TestObservability(t *testing.T) {
	logger := &loggerMock{}
	metrics := &metricsMock{}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	handler := middleware.Observability(logger, metrics)(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/storage/public/missing.png", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}

	if logger.events != 1 {
		t.Fatalf("expected 1 log event, got %d", logger.events)
	}

	if logger.lastEvent.Event != "storage_http_request" {
		t.Fatalf("unexpected event %q", logger.lastEvent.Event)
	}

	if logger.lastEvent.Operation != http.MethodGet {
		t.Fatalf("expected GET operation, got %q", logger.lastEvent.Operation)
	}

	if metrics.incrementCalls != 1 {
		t.Fatalf("expected 1 increment call, got %d", metrics.incrementCalls)
	}

	if metrics.incrementMetric.Name != "storage_http_requests_total" {
		t.Fatalf("unexpected counter %q", metrics.incrementMetric.Name)
	}

	if metrics.incrementMetric.Labels["method"] != http.MethodGet {
		t.Fatalf("expected GET method label, got %q", metrics.incrementMetric.Labels["method"])
	}

	if metrics.incrementMetric.Labels["status"] != "Not Found" {
		t.Fatalf("expected Not Found status label, got %q", metrics.incrementMetric.Labels["status"])
	}

	if metrics.observeCalls != 1 {
		t.Fatalf("expected 1 observe call, got %d", metrics.observeCalls)
	}

	if metrics.observeMetric.Name != "storage_http_request_duration_seconds" {
		t.Fatalf("unexpected duration metric %q", metrics.observeMetric.Name)
	}

	if metrics.observeMetric.Value < 0 {
		t.Fatal("expected non-negative request duration")
	}
}

type loggerMock struct {
	events    int
	lastEvent ports.LogEvent
}

func (l *loggerMock) Log(_ context.Context, event ports.LogEvent) error {
	l.events++
	l.lastEvent = event
	return nil
}

type metricsMock struct {
	incrementCalls  int
	incrementMetric ports.Metric

	observeCalls  int
	observeMetric ports.Metric
}

func (m *metricsMock) Increment(_ context.Context, metric ports.Metric) error {
	m.incrementCalls++
	m.incrementMetric = metric
	return nil
}

func (m *metricsMock) Observe(_ context.Context, metric ports.Metric) error {
	m.observeCalls++
	m.observeMetric = metric
	return nil
}
