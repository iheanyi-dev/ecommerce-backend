package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
)

// contextKey is a private type used to prevent collisions with context
// keys created by other packages.
//
// We never use plain strings as context keys.
type contextKey string

const (
	// authenticatedIdentityKey stores the authenticated user's identity
	// inside the HTTP request context.
	authenticatedIdentityKey contextKey = "authenticated_identity"
)

// AuthenticationMiddleware validates the access token supplied by the
// client and makes the authenticated identity available to downstream
// handlers.
//
// The middleware belongs to Presentation because it operates on HTTP
// requests. JWT-specific validation, however, remains behind the
// application-level TokenService contract.
type AuthenticationMiddleware struct {
	tokenService ports.TokenService
}

// NewAuthenticationMiddleware creates the authentication middleware.
func NewAuthenticationMiddleware(
	tokenService ports.TokenService,
) *AuthenticationMiddleware {
	return &AuthenticationMiddleware{
		tokenService: tokenService,
	}
}

// RequireAuthentication protects an HTTP handler from unauthenticated
// requests.
//
// A valid request must contain:
//
//	Authorization: Bearer <access-token>
//
// The middleware validates the token before allowing the request to reach
// the protected handler.
func (m *AuthenticationMiddleware) RequireAuthentication(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization := strings.TrimSpace(
			r.Header.Get("Authorization"),
		)

		if authorization == "" {
			http.Error(
				w,
				"authorization header is required",
				http.StatusUnauthorized,
			)
			return
		}

		// The Authorization header must use the Bearer authentication
		// scheme.
		parts := strings.Fields(authorization)

		if len(parts) != 2 ||
			!strings.EqualFold(parts[0], "Bearer") {
			http.Error(
				w,
				"invalid authorization header",
				http.StatusUnauthorized,
			)
			return
		}

		tokenString := strings.TrimSpace(parts[1])

		if tokenString == "" {
			http.Error(
				w,
				"invalid authorization header",
				http.StatusUnauthorized,
			)
			return
		}

		identity, err := m.tokenService.ValidateAccessToken(
			r.Context(),
			tokenString,
		)
		if err != nil {
			http.Error(
				w,
				"invalid or expired access token",
				http.StatusUnauthorized,
			)
			return
		}

		// Store only the application-level identity in the request context.
		//
		// The handler does not need to know that the identity originated
		// from a JWT.
		ctx := context.WithValue(
			r.Context(),
			authenticatedIdentityKey,
			identity,
		)

		next.ServeHTTP(
			w,
			r.WithContext(ctx),
		)
	})
}

// AuthenticatedIdentity retrieves the authenticated identity from a request.
//
// The boolean result allows callers to distinguish between:
//
//	identity exists
//
// and:
//
//	identity does not exist
//
// A missing identity normally means that the handler was called without
// the authentication middleware.
func AuthenticatedIdentity(
	ctx context.Context,
) (ports.AuthenticatedIdentity, bool) {
	identity, ok := ctx.Value(
		authenticatedIdentityKey,
	).(ports.AuthenticatedIdentity)

	return identity, ok
}