package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/presentation/middleware"
)

func TestTracing(t *testing.T) {
	tracer := &tracerMock{}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Context().Value(tracingContextKey{}); got != "traced" {
			t.Fatalf("expected traced context value, got %v", got)
		}

		w.WriteHeader(http.StatusNoContent)
	})

	handler := middleware.Tracing(tracer)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}

	if tracer.startCalls != 1 {
		t.Fatalf("expected 1 tracer start call, got %d", tracer.startCalls)
	}

	if tracer.span.endCalls != 1 {
		t.Fatalf("expected span End to be called once, got %d", tracer.span.endCalls)
	}

	if tracer.name != "storage.http.request" {
		t.Fatalf("unexpected span name %q", tracer.name)
	}
}

type tracingContextKey struct{}

type tracerMock struct {
	startCalls int
	name       string
	span       *spanMock
}

func (t *tracerMock) Start(ctx context.Context, name string) (context.Context, ports.Span) {
	t.startCalls++
	t.name = name
	t.span = &spanMock{}

	ctx = context.WithValue(ctx, tracingContextKey{}, "traced")

	return ctx, t.span
}

type spanMock struct {
	endCalls int
}

func (s *spanMock) End() {
	s.endCalls++
}
