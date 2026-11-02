package middleware

import (
	"net/http"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// TracingMiddleware creates one HTTP server span for every Payment request.
//
// Tracing belongs at the HTTP transport boundary. The application layer
// receives the resulting context without depending on OpenTelemetry directly.
//
// Credentials, request bodies, raw URLs, and user-controlled identifiers
// are never included in the span name.
type TracingMiddleware struct {
	tracer ports.Tracer
}

// NewTracingMiddleware creates Payment HTTP tracing middleware.
func NewTracingMiddleware(tracer ports.Tracer) *TracingMiddleware {
	return &TracingMiddleware{
		tracer: tracer,
	}
}

// Middleware extracts an incoming W3C trace context, starts a server span,
// and passes the resulting context downstream.
//
// Tracing is best-effort. A disabled tracer must never prevent a request
// from reaching the Payment router.
func (m *TracingMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m == nil || m.tracer == nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := otel.GetTextMapPropagator().Extract(
			r.Context(),
			propagation.HeaderCarrier(r.Header),
		)

		spanName := r.Method

		if route := strings.TrimSpace(r.Pattern); route != "" {
			if strings.HasPrefix(route, r.Method+" ") {
				route = strings.TrimSpace(
					strings.TrimPrefix(route, r.Method),
				)
			}

			spanName += " " + route
		} else {
			spanName += " payment.request"
		}

		ctx, span := m.tracer.Start(ctx, spanName)
		defer span.End()

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
