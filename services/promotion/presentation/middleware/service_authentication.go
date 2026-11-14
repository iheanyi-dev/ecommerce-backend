package middleware

import (
	"crypto/subtle"
	"net/http"

	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/promotion/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/shared/config"
)

const (
	ServiceNameHeader   = "X-Service-Name"
	ServiceSecretHeader = "X-Service-Secret"
)

type ServiceAuthenticationMiddleware struct {
	serviceName   string
	serviceSecret string
}

func NewServiceAuthenticationMiddleware(cfg *config.Config) *ServiceAuthenticationMiddleware {
	if cfg == nil {
		return &ServiceAuthenticationMiddleware{}
	}
	return &ServiceAuthenticationMiddleware{
		serviceName:   cfg.ServiceAuthName,
		serviceSecret: cfg.ServiceAuthSecret,
	}
}

func (m *ServiceAuthenticationMiddleware) RequireServiceAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m == nil ||
			r.Header.Get(ServiceNameHeader) != m.serviceName ||
			!secureStringEqual(r.Header.Get(ServiceSecretHeader), m.serviceSecret) {
			presentation_errors.WriteError(w, presentation_errors.ErrInvalidServiceCredentials)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func secureStringEqual(left, right string) bool {
	if left == "" || right == "" {
		return false
	}
	leftBytes, rightBytes := []byte(left), []byte(right)
	if len(leftBytes) != len(rightBytes) {
		return false
	}
	return subtle.ConstantTimeCompare(leftBytes, rightBytes) == 1
}
