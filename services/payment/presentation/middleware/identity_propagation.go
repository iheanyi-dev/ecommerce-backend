package middleware

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/payment/presentation/errors"
)

const (
	// AuthenticatedUserIDHeader is populated by the trusted Gateway after
	// successful JWT validation.
	AuthenticatedUserIDHeader = "X-Authenticated-User-ID"

	// AuthenticatedRoleHeader is populated by the trusted Gateway after
	// successful JWT validation.
	AuthenticatedRoleHeader = "X-Authenticated-Role"
)

// IdentityPropagationMiddleware translates the identity established by the
// Gateway into the Payment application's authenticated-identity context.
//
// Payment does not validate user JWTs. The Gateway is the authentication
// boundary; Payment consumes only the trusted identity metadata it propagates.
type IdentityPropagationMiddleware struct{}

// NewIdentityPropagationMiddleware creates Payment identity propagation
// middleware.
func NewIdentityPropagationMiddleware() *IdentityPropagationMiddleware {
	return &IdentityPropagationMiddleware{}
}

// Middleware requires valid trusted identity headers and attaches the
// resulting identity to the request context. Anonymous, incomplete, and
// malformed identities are rejected before reaching the next handler.
func (m *IdentityPropagationMiddleware) Middleware(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userIDHeader := strings.TrimSpace(
			r.Header.Get(AuthenticatedUserIDHeader),
		)
		role := strings.TrimSpace(
			r.Header.Get(AuthenticatedRoleHeader),
		)

		if userIDHeader == "" || role == "" {
			presentation_errors.WriteError(
				w,
				presentation_errors.ErrAuthenticationRequired,
			)
			return
		}

		userID, err := uuid.Parse(userIDHeader)
		if err != nil {
			presentation_errors.WriteError(
				w,
				presentation_errors.ErrAuthenticationRequired,
			)
			return
		}

		identity := ports.AuthenticatedIdentity{
			UserID: userID,
			Role:   role,
		}

		ctx := ports.WithAuthenticatedIdentity(r.Context(), identity)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
