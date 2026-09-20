package middleware

import (
	"net/http"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/gateway/application/ports"
)

const (
	// These headers carry identity established by the Gateway.
	//
	// They are internal identity-context headers and must never be
	// trusted when supplied directly by an external client.
	AuthenticatedUserIDHeader = "X-Authenticated-User-ID"
	AuthenticatedRoleHeader   = "X-Authenticated-Role"

	// Identity Service owns this entire API namespace.
	//
	// Requests under this path must bypass Gateway JWT processing
	// because Identity already performs its own authentication and
	// token validation.
	identityRoutePrefix = "/api/v1/users/"
)

// IdentityPropagationMiddleware optionally authenticates requests
// destined for business services and propagates the resulting
// authenticated identity.
//
// The middleware deliberately does not require a JWT.
//
// Behavior:
//
//   - Identity routes -> forwarded completely untouched.
//   - No JWT on other routes -> forwarded anonymously.
//   - Valid JWT on other routes -> trusted identity is propagated.
//   - Invalid JWT on other routes -> request is rejected.
//
// Downstream business services remain responsible for deciding
// whether authentication is required for a particular endpoint.
type IdentityPropagationMiddleware struct {
	tokenValidator ports.AccessTokenValidator
}

// NewIdentityPropagationMiddleware constructs the middleware using
// the Gateway's access-token validator.
func NewIdentityPropagationMiddleware(
	tokenValidator ports.AccessTokenValidator,
) *IdentityPropagationMiddleware {
	return &IdentityPropagationMiddleware{
		tokenValidator: tokenValidator,
	}
}

// Middleware processes requests entering the Gateway.
func (m *IdentityPropagationMiddleware) Middleware(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Identity owns its own authentication boundary.
		//
		// Do absolutely nothing to Identity requests. This means the
		// request, including Authorization and any other headers,
		// proceeds exactly as received by the Gateway.
		if strings.HasPrefix(r.URL.Path, identityRoutePrefix) {
			next.ServeHTTP(w, r)
			return
		}

		// Never allow an external client to provide trusted identity
		// information to a business service.
		//
		// These headers are removed before examining authentication.
		r.Header.Del(AuthenticatedUserIDHeader)
		r.Header.Del(AuthenticatedRoleHeader)

		authorization := r.Header.Get("Authorization")

		// No JWT means this is an anonymous request.
		//
		// The Gateway does not know whether the downstream endpoint
		// requires authentication, so the request is allowed through.
		if authorization == "" {
			next.ServeHTTP(w, r)
			return
		}

		token, ok := extractBearerToken(authorization)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		identity, err := m.tokenValidator.ValidateAccessToken(
			r.Context(),
			token,
		)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Identity is now derived exclusively from the validated JWT.
		//
		// Set overwrites the values so downstream services receive
		// only the identity established by the Gateway.
		r.Header.Set(AuthenticatedUserIDHeader, identity.UserID)
		r.Header.Set(AuthenticatedRoleHeader, identity.Role)

		next.ServeHTTP(w, r)
	})
}

// extractBearerToken extracts a JWT from:
//
//	Authorization: Bearer <token>
//
// Other authentication schemes are rejected.
func extractBearerToken(authorization string) (string, bool) {
	const bearerPrefix = "Bearer "

	if len(authorization) <= len(bearerPrefix) {
		return "", false
	}

	if !strings.HasPrefix(authorization, bearerPrefix) {
		return "", false
	}

	token := authorization[len(bearerPrefix):]

	if token == "" {
		return "", false
	}

	return token, true
}
