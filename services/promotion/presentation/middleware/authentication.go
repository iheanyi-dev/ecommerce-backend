package middleware

import (
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/promotion/presentation/errors"
)

func RequireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := ports.AuthenticatedIdentityFromContext(r.Context()); !ok {
			presentation_errors.WriteError(w, presentation_errors.ErrAuthenticationRequired)
			return
		}
		next.ServeHTTP(w, r)
	})
}
