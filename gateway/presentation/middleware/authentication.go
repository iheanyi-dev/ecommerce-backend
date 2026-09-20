package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/gateway/application/ports"
)

// contextKey is a private type used to prevent collisions with context keys
// created by other packages.
//
// Plain strings are deliberately not used as context keys.
type contextKey string

const authenticatedIdentityKey contextKey = "authenticated_identity"

// AuthenticationMiddleware validates client access tokens at the Gateway
// boundary.
//
// This middleware is intentionally Gateway-owned.
//
// Business services do not need to parse or validate the client's JWT after
// the Gateway has authenticated the request through the protected
// service-to-service boundary.
type AuthenticationMiddleware struct {
	tokenValidator ports.AccessTokenValidator
}

// NewAuthenticationMiddleware creates Gateway authentication middleware.
func NewAuthenticationMiddleware(
	tokenValidator ports.AccessTokenValidator,
) *AuthenticationMiddleware {
	return &AuthenticationMiddleware{
		tokenValidator: tokenValidator,
	}
}

// RequireAuthentication protects a Gateway route from unauthenticated
// requests.
//
// A protected request must contain:
//
//	Authorization: Bearer <access-token>
//
// After successful validation, only the trusted application-level identity
// is placed into the request context.
func (m *AuthenticationMiddleware) RequireAuthentication(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		authorization := strings.TrimSpace(
			r.Header.Get("Authorization"),
		)

		parts := strings.Fields(authorization)

		if len(parts) != 2 ||
			!strings.EqualFold(parts[0], "Bearer") {
			http.Error(
				w,
				"unauthorized",
				http.StatusUnauthorized,
			)
			return
		}

		identity, err := m.tokenValidator.ValidateAccessToken(
			r.Context(),
			parts[1],
		)
		if err != nil {
			http.Error(
				w,
				"unauthorized",
				http.StatusUnauthorized,
			)
			return
		}

		ctx := WithAuthenticatedIdentity(
			r.Context(),
			identity,
		)

		next.ServeHTTP(
			w,
			r.WithContext(ctx),
		)
	})
}

// WithAuthenticatedIdentity stores the trusted authenticated identity in
// the request context.
//
// The identity originates exclusively from a successfully validated access
// token. It must never be populated from client-controlled identity fields.
func WithAuthenticatedIdentity(
	ctx context.Context,
	identity ports.AuthenticatedIdentity,
) context.Context {
	return context.WithValue(
		ctx,
		authenticatedIdentityKey,
		identity,
	)
}

// AuthenticatedIdentity retrieves the identity established by Gateway
// authentication middleware.
func AuthenticatedIdentity(
	ctx context.Context,
) (ports.AuthenticatedIdentity, bool) {
	identity, ok := ctx.Value(
		authenticatedIdentityKey,
	).(ports.AuthenticatedIdentity)

	return identity, ok
}
