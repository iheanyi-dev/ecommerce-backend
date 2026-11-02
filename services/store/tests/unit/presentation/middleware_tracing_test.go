package presentation_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/middleware"
	"github.com/stretchr/testify/require"
)

type testSpan struct {
	ended bool
}

func (s *testSpan) End() {
	s.ended = true
}

type testTracer struct {
	called  bool
	name    string
	context context.Context
	span    *testSpan
}

func (t *testTracer) Start(ctx context.Context, name string) (context.Context, ports.Span) {
	t.called = true
	t.name = name
	t.context = ctx
	t.span = &testSpan{}

	return context.WithValue(ctx, testTracerContextKey{}, "traced"), t.span
}

type testTracerContextKey struct{}

func TestTracingMiddleware_StartsAndEndsSpan(t *testing.T) {
	tracer := &testTracer{}
	m := middleware.NewTracingMiddleware(tracer)

	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "traced", r.Context().Value(testTracerContextKey{}))
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/stores", nil)

	// ServeMux normally supplies Request.Pattern. The middleware must also
	// behave correctly when invoked directly in isolation.
	request.Pattern = "GET /api/v1/stores"

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.True(t, tracer.called)
	require.Equal(t, "GET /api/v1/stores", tracer.name)
	require.NotNil(t, tracer.span)
	require.True(t, tracer.span.ended)
}

func TestTracingMiddleware_AllowsRequestWhenTracerIsNil(t *testing.T) {
	m := middleware.NewTracingMiddleware(nil)

	called := false

	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/stores", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.True(t, called)
}
