package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
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

		route := strings.TrimSpace(r.Pattern)
		if route == "" {
			route = "order.request"
		} else if strings.HasPrefix(route, r.Method+" ") {
			route = strings.TrimSpace(strings.TrimPrefix(route, r.Method+" "))
		}

		spanName := fmt.Sprintf("%s %s", r.Method, route)
		ctx, span := m.tracer.Start(ctx, spanName)
		if span != nil {
			defer span.End()
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
