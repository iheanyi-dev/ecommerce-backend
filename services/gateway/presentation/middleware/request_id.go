package middleware

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"

// RequestIDMiddleware ensures every Gateway request has a stable request
// identifier.
//
// The identifier is transport metadata used for request correlation across
// the Gateway and downstream services. It does not identify the user and
// must never be treated as an authentication credential.
type RequestIDMiddleware struct{}

// NewRequestIDMiddleware creates request-ID middleware.
func NewRequestIDMiddleware() *RequestIDMiddleware {
	return &RequestIDMiddleware{}
}

// Middleware ensures the request contains a valid UUID request ID and
// returns the same ID in the response.
//
// A valid upstream request ID is preserved. Missing or invalid values are
// replaced with a newly generated UUID.
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

		w.Header().Set(RequestIDHeader, requestID)

		next.ServeHTTP(w, r)
	})
}
