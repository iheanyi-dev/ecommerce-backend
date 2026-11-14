package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/ports"
)

type LogEvent struct {
	Event           string
	Operation       string
	UserID          string
	Role            string
	FailureCategory string
}

type Metric struct {
	Name   string
	Value  float64
	Labels map[string]string
}

type Logger interface {
	Log(context.Context, LogEvent) error
}

type Metrics interface {
	Increment(context.Context, Metric) error
	Observe(context.Context, Metric) error
}

type RequestObservabilityMiddleware struct {
	logger  Logger
	metrics Metrics
}

func NewRequestObservabilityMiddleware(logger Logger, metrics Metrics) *RequestObservabilityMiddleware {
	return &RequestObservabilityMiddleware{logger: logger, metrics: metrics}
}

func (m *RequestObservabilityMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m == nil {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		recorder := newResponseRecorder(w)
		next.ServeHTTP(recorder, r)

		route := r.Pattern
		if route == "" {
			route = "unmatched"
		}
		ctx := r.Context()

		if m.logger != nil {
			event := LogEvent{Event: "http.request.completed", Operation: route}
			if identity, ok := ports.AuthenticatedIdentityFromContext(ctx); ok {
				event.UserID = identity.UserID.String()
				event.Role = identity.Role
			}
			if recorder.statusCode >= http.StatusInternalServerError {
				event.FailureCategory = "server_error"
			} else if recorder.statusCode >= http.StatusBadRequest {
				event.FailureCategory = "client_error"
			}
			_ = m.logger.Log(ctx, event)
		}

		if m.metrics != nil {
			labels := map[string]string{
				"method": r.Method,
				"route":  route,
				"status": strconv.Itoa(recorder.statusCode),
			}
			_ = m.metrics.Increment(ctx, Metric{
				Name: "http.requests", Value: 1, Labels: labels,
			})
			_ = m.metrics.Observe(ctx, Metric{
				Name:   "http.request.duration",
				Value:  float64(time.Since(start).Milliseconds()),
				Labels: map[string]string{"method": r.Method, "route": route},
			})
		}
	})
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
}

func (r *responseRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.statusCode = code
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(body []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(body)
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
