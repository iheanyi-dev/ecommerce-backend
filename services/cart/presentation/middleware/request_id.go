package middleware

import (
	"net/http"

	"github.com/google/uuid"
)

// RequestIDMiddleware preserves a valid request ID or generates one.
type RequestIDMiddleware struct{}

func NewRequestIDMiddleware() *RequestIDMiddleware {
	return &RequestIDMiddleware{}
}

func (m *RequestIDMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if parsed, err := uuid.Parse(id); err != nil || parsed == uuid.Nil {
			id = uuid.NewString()
		}

		r.Header.Set("X-Request-ID", id)
		next.ServeHTTP(w, r)
	})
}
