package middleware

import (
	"net/http"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// TracingMiddleware creates one HTTP server span for every Store request.
//
// Tracing belongs at the HTTP transport boundary. The application layer only
// receives the resulting context and never depends on OpenTelemetry directly.
//
// No credentials, request bodies, raw URLs, or user-controlled identifiers
// are added to the span name.
type TracingMiddleware struct {
	tracer ports.Tracer
}

// NewTracingMiddleware creates Store HTTP tracing middleware.
func NewTracingMiddleware(tracer ports.Tracer) *TracingMiddleware {
	return &TracingMiddleware{
		tracer: tracer,
	}
}

// Middleware extracts an incoming W3C trace context, starts a server span,
// and passes the resulting context downstream.
//
// Tracing is best-effort. A disabled tracer must never prevent a request
// from reaching the Store router.
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
			spanName += " store.request"
		}

		ctx, span := m.tracer.Start(ctx, spanName)
		defer span.End()

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
