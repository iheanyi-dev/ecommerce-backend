package middleware

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
)

const (
	AuthenticatedUserIDHeader = "X-Authenticated-User-ID"
	AuthenticatedRoleHeader   = "X-Authenticated-Role"
)

// IdentityPropagationMiddleware accepts only complete, valid propagated identities.
// Requests without either identity header remain anonymous for public endpoints.
func IdentityPropagationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userIDHeader := strings.TrimSpace(r.Header.Get(AuthenticatedUserIDHeader))
		role := strings.TrimSpace(r.Header.Get(AuthenticatedRoleHeader))

		if userIDHeader == "" && role == "" {
			next.ServeHTTP(w, r)
			return
		}

		if userIDHeader == "" || role == "" {
			writeAuthenticationError(w, http.StatusUnauthorized, "invalid authenticated identity")
			return
		}

		userID, err := uuid.Parse(userIDHeader)
		if err != nil {
			writeAuthenticationError(w, http.StatusUnauthorized, "invalid authenticated identity")
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
