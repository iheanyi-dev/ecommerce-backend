package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/shared/config"
)

const (
	ServiceNameHeader   = "X-Service-Name"
	ServiceSecretHeader = "X-Service-Secret"
)

// NewServiceAuthenticationMiddleware restricts Product HTTP access to trusted services.
func NewServiceAuthenticationMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			serviceName := r.Header.Get(ServiceNameHeader)
			serviceSecret := r.Header.Get(ServiceSecretHeader)

			nameMatches := subtle.ConstantTimeCompare(
				[]byte(serviceName),
				[]byte(cfg.ServiceAuthName),
			)
			secretMatches := subtle.ConstantTimeCompare(
				[]byte(serviceSecret),
				[]byte(cfg.ServiceAuthSecret),
			)

			if serviceName == "" || serviceSecret == "" || nameMatches != 1 || secretMatches != 1 {
				writeAuthenticationError(w, http.StatusUnauthorized, "unauthorized service")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeAuthenticationError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
