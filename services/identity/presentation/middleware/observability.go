package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
)

// RequestObservabilityMiddleware records transport-level observability
// information for every HTTP request.
//
// This middleware deliberately operates at the HTTP boundary. Business-level
// events such as login, registration, password changes, refresh, and logout
// remain the responsibility of the application use cases.
//
// Logging records:
//
//   - HTTP method
//   - matched route pattern
//   - HTTP status code
//   - request duration
//   - authenticated user ID, when available
//   - authenticated role, when available
//   - a coarse failure category for 4xx/5xx responses
//
// Metrics record:
//
//   - completed HTTP request count
//   - HTTP request duration
//
// Metric labels are deliberately limited to stable, low-cardinality transport
// dimensions:
//
//   - method
//   - route
//   - status
//
// The middleware never records request bodies, Authorization headers,
// passwords, password hashes, access tokens, or refresh tokens.
type RequestObservabilityMiddleware struct {
	logger  ports.Logger
	metrics ports.Metrics
}

// NewRequestObservabilityMiddleware creates the HTTP request observability
// middleware.
//
// Logger and metrics are allowed to be nil so that observability remains
// optional in tests and in environments where either capability has
// deliberately been disabled.
func NewRequestObservabilityMiddleware(
	logger ports.Logger,
	metrics ports.Metrics,
) *RequestObservabilityMiddleware {
	return &RequestObservabilityMiddleware{
		logger:  logger,
		metrics: metrics,
	}
}

// Middleware wraps an HTTP handler and records observability information
// after the handler has completed.
//
// Both logging and metrics are best-effort. Observability failures must never
// interfere with the application's HTTP response.
func (m *RequestObservabilityMiddleware) Middleware(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the response writer so that we can observe the final HTTP
		// status code produced by the downstream handler.
		recorder := newResponseRecorder(w)

		// Always allow the actual request to complete normally. Observability
		// is performed afterward and must never interfere with the response.
		next.ServeHTTP(recorder, r)

		duration := time.Since(start)

		m.logRequest(
			r.Context(),
			r,
			recorder.statusCode,
			duration,
		)

		m.recordMetrics(
			r.Context(),
			r,
			recorder.statusCode,
			duration,
		)
	})
}

// logRequest builds and sends the structured request event.
func (m *RequestObservabilityMiddleware) logRequest(
	ctx context.Context,
	r *http.Request,
	statusCode int,
	duration time.Duration,
) {
	if m.logger == nil {
		return
	}

	// Prefer the route pattern assigned by net/http's ServeMux. This gives
	// us a stable route such as "/api/v1/users/{id}" instead of recording
	// potentially high-cardinality values from individual URLs.
	//
	// When the middleware is invoked directly in a unit test or outside a
	// ServeMux, Request.Pattern may be empty. In that case, fall back to
	// the actual URL path.
	route := r.Pattern
	if route == "" {
		route = r.URL.Path
	}

	identity, authenticated := AuthenticatedIdentity(ctx)

	event := ports.LogEvent{
		Event:          "http.request.completed",
		Operation:      route,
		HTTPMethod:     r.Method,
		Route:          route,
		StatusCode:     statusCode,
		DurationMillis: duration.Milliseconds(),
	}

	// Add authenticated identity information only when authentication
	// middleware has already placed it into the request context.
	if authenticated {
		event.UserID = identity.UserID
		event.Role = identity.Role
	}

	// Keep failure categories deliberately coarse at the HTTP transport
	// level. More specific failure reasons are emitted by the application
	// layer where the actual business operation is known.
	switch {
	case statusCode >= http.StatusInternalServerError:
		event.FailureCategory = "server_error"
	case statusCode >= http.StatusBadRequest:
		event.FailureCategory = "client_error"
	}

	// Ignore logger errors. Logging infrastructure must not affect the
	// application's HTTP response.
	_ = m.logger.Log(ctx, event)
}

// recordMetrics records transport-level HTTP metrics.
//
// Metrics deliberately use only stable transport dimensions. User IDs,
// query parameters, request bodies, authorization credentials, and other
// potentially high-cardinality or sensitive values are never used as labels.
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

	// Count every completed HTTP request.
	//
	// A metrics backend failure must never affect the application response.
	_ = m.metrics.Increment(ctx, ports.Metric{
		Name:   "http.requests",
		Value:  1,
		Labels: labels,
	})

	// Record request duration separately so that a metrics backend can later
	// expose latency distributions, averages, percentiles, or SLOs.
	//
	// Duration is represented in milliseconds to remain consistent with the
	// existing structured logging contract.
	durationLabels := map[string]string{
		"method": r.Method,
		"route":  route,
	}

	_ = m.metrics.Observe(ctx, ports.Metric{
		Name:   "http.request.duration",
		Value:  float64(duration.Milliseconds()),
		Labels: durationLabels,
	})
}

// responseRecorder captures the status code written by the downstream
// handler while preserving the underlying response body and headers.
type responseRecorder struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

// newResponseRecorder creates a recorder with HTTP 200 as its default status.
//
// HTTP handlers are allowed to write a body without explicitly calling
// WriteHeader. net/http treats such a response as 200, so our recorder must
// do the same.
func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

// WriteHeader records the first status code written by the handler.
//
// net/http ignores subsequent WriteHeader calls after the first one, so the
// recorder follows the same behavior.
func (r *responseRecorder) WriteHeader(statusCode int) {
	if r.wroteHeader {
		return
	}

	r.statusCode = statusCode
	r.wroteHeader = true

	r.ResponseWriter.WriteHeader(statusCode)
}

// Write records an implicit HTTP 200 response when the handler writes a body
// without explicitly calling WriteHeader.
func (r *responseRecorder) Write(body []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}

	return r.ResponseWriter.Write(body)
}
