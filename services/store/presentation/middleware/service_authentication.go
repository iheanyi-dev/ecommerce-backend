package middleware

import (
	"crypto/subtle"
	"net/http"

	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/shared/config"
)

const (
	ServiceNameHeader   = "X-Service-Name"
	ServiceSecretHeader = "X-Service-Secret"
)

// ServiceAuthenticationMiddleware authenticates trusted internal services.
//
// Store uses this mechanism for service-to-service endpoints such as
// subscription/status operations. It is deliberately separate from user
// authentication because a trusted internal service and an authenticated
// end user represent different security principals.
type ServiceAuthenticationMiddleware struct {
	serviceName   string
	serviceSecret string
}

// NewServiceAuthenticationMiddleware creates middleware using the configured
// Store service credentials.
func NewServiceAuthenticationMiddleware(
	cfg *config.Config,
) *ServiceAuthenticationMiddleware {
	return &ServiceAuthenticationMiddleware{
		serviceName:   cfg.ServiceAuthName,
		serviceSecret: cfg.ServiceAuthSecret,
	}
}

// RequireServiceAuthentication protects an endpoint from unauthenticated
// internal service callers.
func (m *ServiceAuthenticationMiddleware) RequireServiceAuthentication(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serviceName := r.Header.Get(ServiceNameHeader)
		serviceSecret := r.Header.Get(ServiceSecretHeader)

		if serviceName != m.serviceName ||
			!secureStringEqual(serviceSecret, m.serviceSecret) {
			presentation_errors.WriteError(
				w,
				application_errors.ErrInvalidServiceCredentials,
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// secureStringEqual compares secrets in constant time.
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
