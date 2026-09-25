package middleware

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"

// RequestIDMiddleware establishes the correlation identifier used to trace
// one HTTP request across the Gateway and downstream Store operations.
//
// A valid upstream UUID is preserved. Missing or invalid values are replaced
// so every request entering Store has a safe correlation identifier.
type RequestIDMiddleware struct{}

// NewRequestIDMiddleware creates Store request-ID middleware.
func NewRequestIDMiddleware() *RequestIDMiddleware {
	return &RequestIDMiddleware{}
}

// Middleware ensures every request has a valid request ID before any
// downstream middleware, authentication, or handler executes.
func (m *RequestIDMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get(RequestIDHeader))

		if _, err := uuid.Parse(requestID); err != nil {
			requestID = uuid.NewString()
			r.Header.Set(RequestIDHeader, requestID)
		}

		w.Header().Set(RequestIDHeader, requestID)

		next.ServeHTTP(w, r)
	})
}
