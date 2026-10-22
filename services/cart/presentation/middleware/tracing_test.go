package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/ports"
)

type testTracer struct {
	name string
	span *testSpan
}

func (t *testTracer) Start(ctx context.Context, name string) (context.Context, ports.Span) {
	t.name = name
	t.span = &testSpan{}
	return context.WithValue(ctx, testTraceContextKey{}, true), t.span
}

type testTraceContextKey struct{}

type testSpan struct {
	attributes map[string]string
	ints       map[string]int
	status     string
	ended      bool
}

func (s *testSpan) SetAttribute(key, value string) {
	if s.attributes == nil {
		s.attributes = make(map[string]string)
	}
	s.attributes[key] = value
}
func (s *testSpan) SetIntAttribute(key string, value int) {
	if s.ints == nil {
		s.ints = make(map[string]int)
	}
	s.ints[key] = value
}
func (s *testSpan) RecordError(error)                 {}
func (s *testSpan) SetErrorStatus(description string) { s.status = description }
func (s *testSpan) End()                              { s.ended = true }

func TestTracingMiddlewareRecordsRequestAndStatus(t *testing.T) {
	tracer := &testTracer{}
	handler := NewTracingMiddleware(tracer).Middleware(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if got := r.Context().Value(testTraceContextKey{}); got != true {
				t.Fatalf("trace context was not propagated: %v", got)
			}
			w.WriteHeader(http.StatusInternalServerError)
		},
	))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/cart/items", nil)
	handler.ServeHTTP(recorder, request)

	if tracer.name != "cart.http" {
		t.Fatalf("span name = %q, want cart.http", tracer.name)
	}
	if !tracer.span.ended {
		t.Fatal("span was not ended")
	}
	if got := tracer.span.attributes["http.method"]; got != http.MethodPost {
		t.Fatalf("http.method = %q", got)
	}
	if got := tracer.span.attributes["http.target"]; got != "/api/v1/cart/items" {
		t.Fatalf("http.target = %q", got)
	}
	if got := tracer.span.ints["http.status_code"]; got != http.StatusInternalServerError {
		t.Fatalf("http.status_code = %d", got)
	}
	if tracer.span.status == "" {
		t.Fatal("expected error status for 500 response")
	}
}

func TestTracingMiddlewareDefaultsStatusToOK(t *testing.T) {
	tracer := &testTracer{}
	handler := NewTracingMiddleware(tracer).Middleware(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
	)
	handler.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil),
	)

	if got := tracer.span.ints["http.status_code"]; got != http.StatusOK {
		t.Fatalf("http.status_code = %d, want %d", got, http.StatusOK)
	}
	if tracer.span.status != "" {
		t.Fatalf("unexpected error status: %q", tracer.span.status)
	}
}
