package middleware

import (
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/payment/presentation/errors"
)

// RequireAuthentication protects routes that require an authenticated user.
//
// Gateway authentication validates the user's JWT. Payment's identity
// propagation middleware places the resulting trusted identity into the
// request context. This middleware checks only that the identity exists;
// it does not validate JWTs.
func RequireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := ports.AuthenticatedIdentityFromContext(r.Context())
		if !ok {
			presentation_errors.WriteError(
				w,
				presentation_errors.ErrAuthenticationRequired,
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}
