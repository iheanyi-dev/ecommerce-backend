package middleware

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/order/presentation/errors"
)

const (
	AuthenticatedUserIDHeader = "X-Authenticated-User-ID"
	AuthenticatedRoleHeader   = "X-Authenticated-Role"
)

// IdentityPropagationMiddleware translates Gateway-propagated identity into
// application context. It does not validate JWTs or trust partial identities.
type IdentityPropagationMiddleware struct{}

func NewIdentityPropagationMiddleware() *IdentityPropagationMiddleware {
	return &IdentityPropagationMiddleware{}
}

func (m *IdentityPropagationMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userIDHeader := strings.TrimSpace(r.Header.Get(AuthenticatedUserIDHeader))
		role := strings.TrimSpace(r.Header.Get(AuthenticatedRoleHeader))

		if userIDHeader == "" && role == "" {
			next.ServeHTTP(w, r)
			return
		}

		userID, err := uuid.Parse(userIDHeader)
		if userIDHeader == "" || role == "" || err != nil {
			presentation_errors.WriteError(w, presentation_errors.ErrAuthenticationRequired)
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
