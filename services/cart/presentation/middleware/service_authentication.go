package middleware

import (
	"crypto/subtle"
	"net/http"

	presentationerrors "github.com/iheanyi-dev/ecommerce-backend/services/cart/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/shared/config"
)

// ServiceAuthenticationMiddleware protects the Cart service boundary.
type ServiceAuthenticationMiddleware struct {
	name   string
	secret string
}

func NewServiceAuthenticationMiddleware(cfg *config.Config) *ServiceAuthenticationMiddleware {
	return &ServiceAuthenticationMiddleware{
		name:   cfg.ServiceAuthName,
		secret: cfg.ServiceAuthSecret,
	}
}

func (m *ServiceAuthenticationMiddleware) RequireServiceAuthentication(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.Header.Get("X-Service-Name")
		secret := r.Header.Get("X-Service-Secret")

		if m.secret == "" ||
			subtle.ConstantTimeCompare([]byte(name), []byte(m.name)) != 1 ||
			subtle.ConstantTimeCompare([]byte(secret), []byte(m.secret)) != 1 {
			presentationerrors.WriteError(w, presentationerrors.ErrInvalidServiceCredentials)
			return
		}

		next.ServeHTTP(w, r)
	})
}
