package middleware

import (
	"net/http"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/ports"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type TracingMiddleware struct {
	tracer ports.Tracer
}

func NewTracingMiddleware(tracer ports.Tracer) *TracingMiddleware {
	return &TracingMiddleware{tracer: tracer}
}

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
				route = strings.TrimSpace(strings.TrimPrefix(route, r.Method))
			}
			spanName += " " + route
		} else {
			spanName += " promotion.request"
		}

		ctx, span := m.tracer.Start(ctx, spanName)
		defer span.End()

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
