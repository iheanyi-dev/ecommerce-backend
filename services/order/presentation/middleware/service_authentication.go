package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/order/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/shared/config"
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
		serviceName:   strings.TrimSpace(cfg.OrderServiceAuthName),
		serviceSecret: cfg.OrderServiceAuthSecret,
	}
}

func (m *ServiceAuthenticationMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m == nil ||
			m.serviceName == "" ||
			m.serviceSecret == "" ||
			strings.TrimSpace(r.Header.Get(ServiceNameHeader)) != m.serviceName ||
			!constantTimeSecretEqual(r.Header.Get(ServiceSecretHeader), m.serviceSecret) {
			presentation_errors.WriteError(w, presentation_errors.ErrAuthenticationRequired)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func constantTimeSecretEqual(provided, expected string) bool {
	if provided == "" || expected == "" || len(provided) != len(expected) {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}
