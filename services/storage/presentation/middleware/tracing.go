package middleware

import (
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
)

const storageHTTPSpanName = "storage.http.request"

// Tracing creates a presentation-layer span for each Storage HTTP request.
func Tracing(tracer ports.Tracer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracer.Start(r.Context(), storageHTTPSpanName)
			defer span.End()

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
