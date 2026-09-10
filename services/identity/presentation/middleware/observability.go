package middleware

import (
	"context"
	"net/http"
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
// The middleware records:
//
//   - HTTP method
//   - matched route pattern
//   - HTTP status code
//   - request duration
//   - authenticated user ID, when available
//   - authenticated role, when available
//   - a coarse failure category for 4xx/5xx responses
//
// It never records request bodies, Authorization headers, passwords,
// password hashes, access tokens, or refresh tokens.
type RequestObservabilityMiddleware struct {
	logger ports.Logger
}

// NewRequestObservabilityMiddleware creates the HTTP request observability
// middleware.
//
// The logger is allowed to be nil so that observability remains optional in
// tests and in environments where logging has deliberately been disabled.
func NewRequestObservabilityMiddleware(
	logger ports.Logger,
) *RequestObservabilityMiddleware {
	return &RequestObservabilityMiddleware{
		logger: logger,
	}
}

// Middleware wraps an HTTP handler and records one structured observability
// event after the handler has completed.
func (m *RequestObservabilityMiddleware) Middleware(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the response writer so that we can observe the final HTTP
		// status code produced by the downstream handler.
		recorder := newResponseRecorder(w)

		// Always allow the actual request to complete normally. Logging is
		// performed afterward and must never interfere with the response.
		next.ServeHTTP(recorder, r)

		// Logging is intentionally best-effort. Observability must never turn
		// a successful request into a failed request.
		m.logRequest(r.Context(), r, recorder.statusCode, time.Since(start))
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
