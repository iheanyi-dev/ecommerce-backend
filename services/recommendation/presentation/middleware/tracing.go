package middleware

import (
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/ports"
)

// Tracing creates a tracing span for each HTTP request.
type Tracing struct {
	tracer ports.Tracer
}

// NewTracing creates tracing middleware.
func NewTracing(tracer ports.Tracer) *Tracing {
	return &Tracing{tracer: tracer}
}

func (m *Tracing) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.tracer == nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx, span := m.tracer.Start(r.Context(), r.Method+" "+r.URL.Path)
		defer span.End()

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
