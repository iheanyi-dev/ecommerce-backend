package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/presentation/middleware"
)

type tracingTestTracer struct {
	spanName string
	traceID  string
	span     *tracingTestSpan
}

func (t *tracingTestTracer) Start(
	ctx context.Context,
	name string,
) (context.Context, ports.Span) {
	t.spanName = name
	t.traceID = trace.SpanContextFromContext(ctx).TraceID().String()
	t.span = &tracingTestSpan{}

	return ctx, t.span
}

type tracingTestSpan struct {
	ended bool
}

func (s *tracingTestSpan) End() {
	s.ended = true
}

var (
	_ ports.Tracer = (*tracingTestTracer)(nil)
	_ ports.Span   = (*tracingTestSpan)(nil)
)

func TestTracingMiddlewareExtractsContextAndCreatesRouteSpan(t *testing.T) {
	previousPropagator := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otel.SetTextMapPropagator(previousPropagator)
	})

	tracer := &tracingTestTracer{}
	middlewareUnderTest := middleware.NewTracingMiddleware(tracer)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(
			t,
			"4bf92f3577b34da6a3ce929d0e0e4736",
			trace.SpanContextFromContext(r.Context()).TraceID().String(),
		)
		w.WriteHeader(http.StatusNoContent)
	})

	handler := middlewareUnderTest.Middleware(next)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/payments/123", nil)
	request.Pattern = "POST /api/v1/payments/{paymentID}"
	request.Header.Set(
		"traceparent",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
	)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusNoContent, response.Code)
	require.Equal(t, "POST /api/v1/payments/{paymentID}", tracer.spanName)
	require.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", tracer.traceID)
	require.NotNil(t, tracer.span)
	require.True(t, tracer.span.ended)
}

func TestTracingMiddlewareUsesFallbackSpanNameWithoutRoutePattern(t *testing.T) {
	tracer := &tracingTestTracer{}
	middlewareUnderTest := middleware.NewTracingMiddleware(tracer)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/payments", nil)
	response := httptest.NewRecorder()

	middlewareUnderTest.Middleware(next).ServeHTTP(response, request)

	require.Equal(t, http.StatusNoContent, response.Code)
	require.Equal(t, "GET payment.request", tracer.spanName)
	require.NotNil(t, tracer.span)
	require.True(t, tracer.span.ended)
}

func TestTracingMiddlewareContinuesWhenTracerIsNil(t *testing.T) {
	tests := []struct {
		name       string
		middleware *middleware.TracingMiddleware
	}{
		{
			name:       "nil tracer",
			middleware: middleware.NewTracingMiddleware(nil),
		},
		{
			name:       "nil middleware",
			middleware: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			})

			request := httptest.NewRequest(http.MethodGet, "/api/v1/payments", nil)
			response := httptest.NewRecorder()

			tt.middleware.Middleware(next).ServeHTTP(response, request)

			require.True(t, called)
			require.Equal(t, http.StatusNoContent, response.Code)
		})
	}
}
