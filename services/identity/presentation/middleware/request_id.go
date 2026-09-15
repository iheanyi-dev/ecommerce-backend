package middleware

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"

// RequestIDMiddleware ensures every Identity HTTP request has a stable
// request identifier.
//
// The request ID is transport metadata only. It identifies the HTTP request
// for logging, tracing, and cross-service correlation. It must never be used
// as a user identity or authentication credential.
type RequestIDMiddleware struct{}

// NewRequestIDMiddleware creates request-ID middleware.
func NewRequestIDMiddleware() *RequestIDMiddleware {
	return &RequestIDMiddleware{}
}

// Middleware ensures the request contains a valid request ID and returns the
// same ID in the HTTP response.
//
// Valid UUID request IDs supplied by an upstream service are preserved.
// Missing or invalid IDs are replaced with a newly generated UUID.
func (m *RequestIDMiddleware) Middleware(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(
			r.Header.Get(RequestIDHeader),
		)

		if _, err := uuid.Parse(requestID); err != nil {
			requestID = uuid.NewString()
			r.Header.Set(RequestIDHeader, requestID)
		}

		next.ServeHTTP(w, r)
	})
}
