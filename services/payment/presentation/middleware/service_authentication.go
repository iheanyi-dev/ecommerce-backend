package middleware

import (
	"crypto/subtle"
	"net/http"

	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/payment/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/shared/config"
)

const (
	ServiceNameHeader   = "X-Service-Name"
	ServiceSecretHeader = "X-Service-Secret"
)

// ServiceAuthenticationMiddleware authenticates trusted internal services.
// It is separate from user authentication and identity propagation.
type ServiceAuthenticationMiddleware struct {
	serviceName   string
	serviceSecret string
}

// NewServiceAuthenticationMiddleware creates service authentication middleware
// using the Payment service's configured credentials.
func NewServiceAuthenticationMiddleware(cfg *config.Config) *ServiceAuthenticationMiddleware {
	return &ServiceAuthenticationMiddleware{
		serviceName:   cfg.ServiceAuthName,
		serviceSecret: cfg.ServiceAuthSecret,
	}
}

// RequireServiceAuthentication rejects requests without valid internal-service
// credentials.
func (m *ServiceAuthenticationMiddleware) RequireServiceAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serviceName := r.Header.Get(ServiceNameHeader)
		serviceSecret := r.Header.Get(ServiceSecretHeader)

		if serviceName != m.serviceName ||
			!secureStringEqual(serviceSecret, m.serviceSecret) {
			presentation_errors.WriteError(
				w,
				presentation_errors.ErrInvalidServiceCredentials,
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// secureStringEqual compares non-empty credentials in constant time.
func secureStringEqual(left, right string) bool {
	if left == "" || right == "" {
		return false
	}

	leftBytes := []byte(left)
	rightBytes := []byte(right)

	if len(leftBytes) != len(rightBytes) {
		return false
	}

	return subtle.ConstantTimeCompare(leftBytes, rightBytes) == 1
}
