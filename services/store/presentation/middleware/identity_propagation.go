package middleware

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
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
// Gateway into the Store application's authenticated-identity context.
//
// Store deliberately does not validate user JWTs. The Gateway is the
// authentication boundary; Store only consumes the trusted identity metadata
// propagated across the internal service boundary.
type IdentityPropagationMiddleware struct{}

// NewIdentityPropagationMiddleware creates Store identity propagation
// middleware.
func NewIdentityPropagationMiddleware() *IdentityPropagationMiddleware {
	return &IdentityPropagationMiddleware{}
}

// Middleware reads the trusted Gateway identity headers and attaches a valid
// authenticated identity to the request context.
//
// Missing identity headers mean the request is anonymous. Malformed or
// incomplete identity headers are rejected rather than silently treated as
// anonymous, because an invalid trusted identity should never cross into the
// application layer.
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

		// No propagated identity means anonymous access.
		if userIDHeader == "" && role == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Once one identity header is present, both are required.
		if userIDHeader == "" || role == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		userID, err := uuid.Parse(userIDHeader)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		identity := ports.AuthenticatedIdentity{
			UserID: userID,
			Role:   role,
		}

		ctx := ports.WithAuthenticatedIdentity(
			r.Context(),
			identity,
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
