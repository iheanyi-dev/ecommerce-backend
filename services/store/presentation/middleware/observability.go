package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
)

// RequestObservabilityMiddleware records safe transport-level HTTP
// observability for every Store request.
//
// Business events remain the responsibility of Store application use cases.
// This middleware records only HTTP-level information.
type RequestObservabilityMiddleware struct {
	logger  ports.Logger
	metrics ports.Metrics
}

// NewRequestObservabilityMiddleware creates Store HTTP observability
// middleware.
//
// Logger and metrics are optional so isolated tests and deliberately
// disabled observability environments continue to work.
func NewRequestObservabilityMiddleware(
	logger ports.Logger,
	metrics ports.Metrics,
) *RequestObservabilityMiddleware {
	return &RequestObservabilityMiddleware{
		logger:  logger,
		metrics: metrics,
	}
}

// Middleware records the final HTTP status and duration after the downstream
// request has completed.
//
// Observability failures are deliberately ignored and can never alter the
// HTTP response.
func (m *RequestObservabilityMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := newResponseRecorder(w)

		next.ServeHTTP(recorder, r)

		duration := time.Since(start)

		m.logRequest(
			r.Context(),
			r,
			recorder.statusCode,
		)

		m.recordMetrics(
			r.Context(),
			r,
			recorder.statusCode,
			duration,
		)
	})
}

func (m *RequestObservabilityMiddleware) logRequest(
	ctx context.Context,
	r *http.Request,
	statusCode int,
) {
	if m.logger == nil {
		return
	}

	route := r.Pattern
	if route == "" {
		route = r.URL.Path
	}

	event := ports.LogEvent{
		Event:     "http.request.completed",
		Operation: route,
	}

	if identity, ok := ports.AuthenticatedIdentityFromContext(ctx); ok {
		event.UserID = identity.UserID.String()
		event.Role = identity.Role
	}

	switch {
	case statusCode >= http.StatusInternalServerError:
		event.FailureCategory = "server_error"
	case statusCode >= http.StatusBadRequest:
		event.FailureCategory = "client_error"
	}

	_ = m.logger.Log(ctx, event)
}

func (m *RequestObservabilityMiddleware) recordMetrics(
	ctx context.Context,
	r *http.Request,
	statusCode int,
	duration time.Duration,
) {
	if m.metrics == nil {
		return
	}

	route := r.Pattern
	if route == "" {
		route = r.URL.Path
	}

	labels := map[string]string{
		"method": r.Method,
		"route":  route,
		"status": strconv.Itoa(statusCode),
	}

	_ = m.metrics.Increment(ctx, ports.Metric{
		Name:   "http.requests",
		Value:  1,
		Labels: labels,
	})

	_ = m.metrics.Observe(ctx, ports.Metric{
		Name:  "http.request.duration",
		Value: float64(duration.Milliseconds()),
		Labels: map[string]string{
			"method": r.Method,
			"route":  route,
		},
	})
}

// responseRecorder captures the final HTTP status while preserving the
// streaming capabilities required by handlers such as ChatHandler.
//
// In particular, ChatHandler requires http.Flusher for Server-Sent Events.
// A middleware wrapper that does not expose Flusher would silently break
// streaming even though the underlying ResponseWriter supports it.
type responseRecorder struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	if r.wroteHeader {
		return
	}

	r.statusCode = statusCode
	r.wroteHeader = true

	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(body []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}

	return r.ResponseWriter.Write(body)
}

// Flush preserves http.Flusher for streaming HTTP handlers.
//
// Store's ChatHandler uses Server-Sent Events and explicitly requires
// http.Flusher to send each generated event to the client immediately.
func (r *responseRecorder) Flush() {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}

	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Unwrap exposes the original ResponseWriter to code that intentionally
// needs access to the underlying HTTP writer.
func (r *responseRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}
