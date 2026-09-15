package handlers

import (
	"encoding/json"
	nethttp "net/http"
)

// HealthHandler handles Gateway health checks.
//
// The health endpoint intentionally has no dependency on downstream
// services. It reports whether the Gateway process itself is able to
// accept and handle HTTP requests.
type HealthHandler struct{}

// NewHealthHandler creates a new Gateway health handler.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// ServeHTTP implements net/http.Handler.
func (h *HealthHandler) ServeHTTP(
	w nethttp.ResponseWriter,
	_ *nethttp.Request,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(nethttp.StatusOK)

	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}
