// services/identity/presentation/middleware/service_authentication.go

package middleware

import (
	"crypto/subtle"
	"net/http"

	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/shared/config"
)

const (
	serviceNameHeader   = "X-Service-Name"
	serviceSecretHeader = "X-Service-Secret"
)

// ServiceAuthenticationMiddleware authenticates trusted internal services.
//
// This authentication mechanism is intentionally separate from user JWT
// authentication. A service credential answers:
//
//	"Which internal service is calling Identity?"
//
// The existing JWT middleware answers:
//
//	"Which user is making this request?"
//
// Both mechanisms may therefore participate in the same request.
type ServiceAuthenticationMiddleware struct {
	serviceName   string
	serviceSecret string
}

// NewServiceAuthenticationMiddleware creates the middleware using the
// configured trusted service identity.
func NewServiceAuthenticationMiddleware(
	cfg *config.Config,
) *ServiceAuthenticationMiddleware {
	return &ServiceAuthenticationMiddleware{
		serviceName:   cfg.ServiceAuthName,
		serviceSecret: cfg.ServiceAuthSecret,
	}
}

// RequireServiceAuthentication rejects requests that do not originate from
// the configured trusted internal service.
func (m *ServiceAuthenticationMiddleware) RequireServiceAuthentication(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		serviceName := r.Header.Get(serviceNameHeader)
		serviceSecret := r.Header.Get(serviceSecretHeader)

		if serviceName != m.serviceName ||
			!secureStringEqual(serviceSecret, m.serviceSecret) {
			errors.WriteError(
				w,
				application_errors.ErrInvalidServiceCredentials,
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// secureStringEqual compares secrets in constant time so the comparison does
// not expose the secret through ordinary timing differences.
func secureStringEqual(left string, right string) bool {
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
