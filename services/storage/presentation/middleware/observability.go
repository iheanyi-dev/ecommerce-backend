package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
)

type observabilityResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *observabilityResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *observabilityResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}

	return w.ResponseWriter.Write(body)
}

// Observability records safe request-level logging and metrics.
//
// It deliberately does not log service credentials, object references,
// object contents, or authorization tokens.
func Observability(logger ports.Logger, metrics ports.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			writer := &observabilityResponseWriter{
				ResponseWriter: w,
			}

			next.ServeHTTP(writer, r)

			status := writer.status
			if status == 0 {
				status = http.StatusOK
			}

			ctx := r.Context()

			_ = logger.Log(ctx, ports.LogEvent{
				Event:     "storage_http_request",
				Operation: r.Method,
			})

			_ = metrics.Increment(ctx, ports.Metric{
				Name:  "storage_http_requests_total",
				Value: 1,
				Labels: map[string]string{
					"method": r.Method,
					"status": http.StatusText(status),
				},
			})

			_ = metrics.Observe(ctx, ports.Metric{
				Name:  "storage_http_request_duration_seconds",
				Value: time.Since(start).Seconds(),
				Labels: map[string]string{
					"method": r.Method,
				},
			})
		})
	}
}

// Keep context usage explicit at the middleware boundary.
var _ context.Context
