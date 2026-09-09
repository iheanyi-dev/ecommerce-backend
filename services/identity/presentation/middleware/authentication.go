package middleware

import (
	"context"
	"net/http"
	"strings"

	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/errors"
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

// WithAuthenticatedIdentity stores an authenticated identity in the request
// context.
//
// This helper keeps context-key access inside the middleware package
// instead of exposing the private context key to other packages.
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
			// Return authentication failures through the common presentation
			// error translator so the API uses one consistent JSON format.
			presentation_errors.WriteError(
				w,
				application_errors.ErrInvalidAccessToken,
			)
			return
		}

		// The Authorization header must use the Bearer authentication
		// scheme.
		parts := strings.Fields(authorization)

		if len(parts) != 2 ||
			!strings.EqualFold(parts[0], "Bearer") {
			// Malformed authorization headers are authentication failures and
			// therefore use the centralized presentation error response.
			presentation_errors.WriteError(
				w,
				application_errors.ErrInvalidAccessToken,
			)
			return
		}

		tokenString := strings.TrimSpace(parts[1])

		if tokenString == "" {
			// A missing bearer token is treated as an invalid access token and
			// translated through the common presentation error mechanism.
			presentation_errors.WriteError(
				w,
				application_errors.ErrInvalidAccessToken,
			)
			return
		}

		identity, err := m.tokenService.ValidateAccessToken(
			r.Context(),
			tokenString,
		)
		if err != nil {
			// Delegate token-validation failures to the common presentation
			// error translator instead of returning a plain-text HTTP error.
			presentation_errors.WriteError(w, err)
			return
		}

		// Store only the application-level identity in the request context.
		//
		// The handler does not need to know that the identity originated
		// from a JWT.
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
