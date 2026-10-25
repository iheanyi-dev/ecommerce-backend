package middleware

import (
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/order/presentation/errors"
)

// RequireAuthentication requires an identity established by trusted middleware.
// JWT validation remains the Gateway's responsibility.
func RequireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := ports.AuthenticatedIdentityFromContext(r.Context())
		if !ok || identity.UserID == [16]byte{} || identity.Role == "" {
			presentation_errors.WriteError(w, presentation_errors.ErrAuthenticationRequired)
			return
		}

		next.ServeHTTP(w, r)
	})
}
