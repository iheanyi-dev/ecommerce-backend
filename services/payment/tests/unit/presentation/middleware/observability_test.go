package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/presentation/middleware"
)

type recordingLogger struct {
	events []ports.LogEvent
	err    error
}

func (l *recordingLogger) Log(_ context.Context, event ports.LogEvent) error {
	l.events = append(l.events, event)
	return l.err
}

type recordingMetrics struct {
	increments   []ports.Metric
	observations []ports.Metric
	err          error
}

func (m *recordingMetrics) Increment(_ context.Context, metric ports.Metric) error {
	m.increments = append(m.increments, metric)
	return m.err
}

func (m *recordingMetrics) Observe(_ context.Context, metric ports.Metric) error {
	m.observations = append(m.observations, metric)
	return m.err
}

func TestRequestObservabilityRecordsRequest(t *testing.T) {
	logger := &recordingLogger{}
	metrics := &recordingMetrics{}

	handler := middleware.NewRequestObservabilityMiddleware(logger, metrics).
		Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/payments", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}

	if len(logger.events) != 1 {
		t.Fatalf("expected one log event, got %d", len(logger.events))
	}

	event := logger.events[0]
	if event.Event != "http.request.completed" {
		t.Fatalf("unexpected event name: %q", event.Event)
	}
	if event.Operation != "/payments" {
		t.Fatalf("expected operation /payments, got %q", event.Operation)
	}

	if len(metrics.increments) != 1 {
		t.Fatalf("expected one increment metric, got %d", len(metrics.increments))
	}
	if metrics.increments[0].Name != "http.requests" {
		t.Fatalf("unexpected increment metric: %q", metrics.increments[0].Name)
	}
	if metrics.increments[0].Labels["status"] != "201" {
		t.Fatalf("expected status label 201, got %q", metrics.increments[0].Labels["status"])
	}

	if len(metrics.observations) != 1 {
		t.Fatalf("expected one duration observation, got %d", len(metrics.observations))
	}
	if metrics.observations[0].Name != "http.request.duration" {
		t.Fatalf("unexpected duration metric: %q", metrics.observations[0].Name)
	}
}

func TestRequestObservabilityUsesRoutePattern(t *testing.T) {
	logger := &recordingLogger{}
	metrics := &recordingMetrics{}

	handler := middleware.NewRequestObservabilityMiddleware(logger, metrics).
		Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Pattern = "POST /payments/{paymentID}"
			w.WriteHeader(http.StatusOK)
		}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/payments/123", nil)
	handler.ServeHTTP(recorder, request)

	if logger.events[0].Operation != "POST /payments/{paymentID}" {
		t.Fatalf("expected route pattern in log, got %q", logger.events[0].Operation)
	}
	if metrics.increments[0].Labels["route"] != "POST /payments/{paymentID}" {
		t.Fatalf("expected route pattern in metrics, got %q", metrics.increments[0].Labels["route"])
	}
}

func TestRequestObservabilityRecordsAuthenticatedIdentity(t *testing.T) {
	logger := &recordingLogger{}
	metrics := &recordingMetrics{}

	handler := middleware.NewRequestObservabilityMiddleware(logger, metrics).
		Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

	request := httptest.NewRequest(http.MethodGet, "/payments", nil)
	identity := ports.AuthenticatedIdentity{
		UserID: uuidForTest(t),
		Role:   "customer",
	}
	ctx := ports.WithAuthenticatedIdentity(request.Context(), identity)

	handler.ServeHTTP(httptest.NewRecorder(), request.WithContext(ctx))

	if logger.events[0].UserID != identity.UserID.String() {
		t.Fatalf("expected user ID %q, got %q", identity.UserID, logger.events[0].UserID)
	}
	if logger.events[0].Role != identity.Role {
		t.Fatalf("expected role %q, got %q", identity.Role, logger.events[0].Role)
	}
}

func TestRequestObservabilityClassifiesFailures(t *testing.T) {
	tests := []struct {
		name            string
		status          int
		expectedFailure string
	}{
		{name: "client error", status: http.StatusBadRequest, expectedFailure: "client_error"},
		{name: "server error", status: http.StatusInternalServerError, expectedFailure: "server_error"},
		{name: "successful response", status: http.StatusOK, expectedFailure: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger := &recordingLogger{}
			handler := middleware.NewRequestObservabilityMiddleware(logger, nil).
				Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(test.status)
				}))

			handler.ServeHTTP(
				httptest.NewRecorder(),
				httptest.NewRequest(http.MethodGet, "/", nil),
			)

			if got := logger.events[0].FailureCategory; got != test.expectedFailure {
				t.Fatalf("expected failure category %q, got %q", test.expectedFailure, got)
			}
		})
	}
}

func TestRequestObservabilityAllowsNilDependencies(t *testing.T) {
	handler := middleware.NewRequestObservabilityMiddleware(nil, nil).
		Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
}

func TestRequestObservabilityIgnoresDependencyFailures(t *testing.T) {
	logger := &recordingLogger{err: errors.New("logging unavailable")}
	metrics := &recordingMetrics{err: errors.New("metrics unavailable")}

	handler := middleware.NewRequestObservabilityMiddleware(logger, metrics).
		Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
		}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, recorder.Code)
	}
}

func TestResponseRecorderDefaultsToOK(t *testing.T) {
	recorder := httptest.NewRecorder()
	wrapped := middleware.NewRequestObservabilityMiddleware(nil, nil).
		Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("ok"))
		}))

	wrapped.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if recorder.Body.String() != "ok" {
		t.Fatalf("expected response body %q, got %q", "ok", recorder.Body.String())
	}
}

func TestResponseRecorderPreservesFlusher(t *testing.T) {
	flushed := false
	recorder := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	recorder.onFlush = func() {
		flushed = true
	}

	handler := middleware.NewRequestObservabilityMiddleware(nil, nil).
		Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatal("expected response writer to implement http.Flusher")
			}
			flusher.Flush()
		}))

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if !flushed {
		t.Fatal("expected underlying response writer to be flushed")
	}
}

type flushRecorder struct {
	*httptest.ResponseRecorder
	onFlush func()
}

func (r *flushRecorder) Flush() {
	r.onFlush()
	r.ResponseRecorder.Flush()
}

func uuidForTest(t *testing.T) uuid.UUID {
	t.Helper()
	return uuid.New()
}
