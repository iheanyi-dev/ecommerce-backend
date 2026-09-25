package middleware

import (
	"net/http"

	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/errors"
)

// RequireAuthentication protects a route that requires an authenticated user.
//
// Gateway authentication is responsible for validating the user's JWT and
// Store's identity-propagation middleware places the resulting trusted
// identity into the request context. This middleware only enforces that the
// identity exists; it does not perform JWT validation again.
func RequireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := ports.AuthenticatedIdentityFromContext(r.Context())
		if !ok {
			presentation_errors.WriteError(
				w,
				application_errors.ErrUnauthenticated,
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}
