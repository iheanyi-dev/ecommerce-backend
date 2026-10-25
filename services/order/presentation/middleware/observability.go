package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
)

type RequestObservabilityMiddleware struct {
	logger  ports.Logger
	metrics ports.Metrics
}

func NewRequestObservabilityMiddleware(
	logger ports.Logger,
	metrics ports.Metrics,
) *RequestObservabilityMiddleware {
	return &RequestObservabilityMiddleware{
		logger:  logger,
		metrics: metrics,
	}
}

func (m *RequestObservabilityMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &responseRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(recorder, r)

		duration := time.Since(started)
		route := r.Pattern
		if route == "" {
			route = r.URL.Path
		}
		status := recorder.status
		statusText := strconv.Itoa(status)

		event := ports.LogEvent{
			Event:     "http.request.completed",
			Operation: route,
		}
		if identity, ok := ports.AuthenticatedIdentityFromContext(r.Context()); ok {
			event.UserID = identity.UserID.String()
			event.Role = identity.Role
		}
		if status >= http.StatusInternalServerError {
			event.FailureCategory = "server_error"
		} else if status >= http.StatusBadRequest {
			event.FailureCategory = "client_error"
		}

		if m != nil && m.logger != nil {
			_ = m.logger.Log(r.Context(), event)
		}
		if m != nil && m.metrics != nil {
			labels := map[string]string{
				"method": r.Method,
				"route":  route,
				"status": statusText,
			}
			_ = m.metrics.Increment(r.Context(), ports.Metric{
				Name:   "http.requests",
				Value:  1,
				Labels: labels,
			})
			_ = m.metrics.Observe(r.Context(), ports.Metric{
				Name:  "http.request.duration",
				Value: float64(duration.Milliseconds()),
				Labels: map[string]string{
					"method": r.Method,
					"route":  route,
				},
			})
		}
	})
}

type responseRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *responseRecorder) WriteHeader(status int) {
	if r.wroteHeader {
		return
	}
	r.status = status
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(data)
}

func (r *responseRecorder) Flush() {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (r *responseRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}
