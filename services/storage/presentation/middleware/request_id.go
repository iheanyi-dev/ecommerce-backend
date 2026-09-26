package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

const RequestIDHeader = "X-Request-ID"

// RequestID ensures every Storage request has a correlation identifier.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(RequestIDHeader)

		if requestID == "" {
			requestID = newRequestID()
		}

		w.Header().Set(RequestIDHeader, requestID)
		next.ServeHTTP(w, r)
	})
}

func newRequestID() string {
	var bytes [16]byte

	if _, err := rand.Read(bytes[:]); err != nil {
		// This is only a correlation identifier. A deterministic fallback
		// keeps request processing available if the system RNG fails.
		return "storage-request"
	}

	return hex.EncodeToString(bytes[:])
}
