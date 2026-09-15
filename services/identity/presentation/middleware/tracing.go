package middleware

import (
	"net/http"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// TracingMiddleware creates a server span for each HTTP request.
//
// The middleware is deliberately kept at the HTTP transport boundary. It
// extracts an incoming W3C trace context, creates a server span, places the
// resulting context into the request, and allows downstream application code
// to continue the same trace.
//
// No credentials, tokens, request bodies, or arbitrary URL values are added
// to the span by this middleware.
type TracingMiddleware struct {
	tracer ports.Tracer
}

// NewTracingMiddleware creates HTTP tracing middleware.
//
// The tracer may be nil so isolated tests and environments that deliberately
// disable tracing can continue to operate normally.
func NewTracingMiddleware(
	tracer ports.Tracer,
) *TracingMiddleware {
	return &TracingMiddleware{
		tracer: tracer,
	}
}

// Middleware wraps an HTTP handler with distributed tracing.
//
// Incoming trace context is extracted before creating the server span. The
// resulting context is then attached to the request so authentication,
// authorization, handlers, and application use cases can continue the same
// trace.
//
// Tracing is best-effort. If tracing is unavailable, the request continues
// normally.
func (m *TracingMiddleware) Middleware(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m == nil || m.tracer == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Extract the trace context supplied by an upstream service or
		// client using the globally configured OpenTelemetry propagator.
		//
		// This allows a future API gateway or another microservice to
		// propagate the same distributed trace into Identity.
		propagator := otel.GetTextMapPropagator()

		ctx := propagator.Extract(
			r.Context(),
			propagation.HeaderCarrier(r.Header),
		)

		// Prefer the stable ServeMux route pattern over the raw URL path.
		// The raw path may contain high-cardinality identifiers.
		spanName := r.Method
		if route := strings.TrimSpace(r.Pattern); route != "" {
			if strings.HasPrefix(route, r.Method+" ") {
				route = strings.TrimSpace(
					strings.TrimPrefix(route, r.Method),
				)
			}

			spanName += " " + route
		} else {
			spanName += " identity.request"
		}

		ctx, span := m.tracer.Start(ctx, spanName)
		defer span.End()

		next.ServeHTTP(
			w,
			r.WithContext(ctx),
		)
	})
}
