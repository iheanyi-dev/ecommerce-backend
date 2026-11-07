package middleware

import (
	"crypto/subtle"
	"net/http"

	presentationerrors "github.com/iheanyi-dev/ecommerce-backend/services/recommendation/presentation/errors"
)

// ServiceAuthentication validates service-to-service authentication headers.
type ServiceAuthentication struct {
	serviceName   string
	serviceSecret string
}

// NewServiceAuthentication creates service authentication middleware.
func NewServiceAuthentication(
	serviceName string,
	serviceSecret string,
) *ServiceAuthentication {
	return &ServiceAuthentication{
		serviceName:   serviceName,
		serviceSecret: serviceSecret,
	}
}

func (m *ServiceAuthentication) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providedName := r.Header.Get("X-Service-Name")
		providedSecret := r.Header.Get("X-Service-Secret")

		nameValid := providedName == m.serviceName
		secretValid := subtle.ConstantTimeCompare(
			[]byte(providedSecret),
			[]byte(m.serviceSecret),
		) == 1

		if !nameValid || !secretValid {
			presentationerrors.WriteError(
				w,
				http.StatusUnauthorized,
				"unauthorized",
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}
