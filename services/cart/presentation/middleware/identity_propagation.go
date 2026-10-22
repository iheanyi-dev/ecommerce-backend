package middleware

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/ports"
	presentationerrors "github.com/iheanyi-dev/ecommerce-backend/services/cart/presentation/errors"
)

// IdentityPropagationMiddleware accepts identity headers only at the trusted
// internal service boundary. Service authentication must wrap this middleware.
type IdentityPropagationMiddleware struct{}

func NewIdentityPropagationMiddleware() *IdentityPropagationMiddleware {
	return &IdentityPropagationMiddleware{}
}

func (m *IdentityPropagationMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-Authenticated-User-ID")
		role := r.Header.Get("X-Authenticated-Role")

		if userID == "" && role == "" {
			next.ServeHTTP(w, r)
			return
		}

		id, err := uuid.Parse(userID)
		if err != nil || id == uuid.Nil || role == "" {
			presentationerrors.WriteError(w, presentationerrors.ErrAuthenticationRequired)
			return
		}

		ctx := ports.WithAuthenticatedIdentity(r.Context(), ports.AuthenticatedIdentity{
			UserID: id,
			Role:   role,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
