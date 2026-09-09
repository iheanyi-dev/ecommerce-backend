package middleware

import (
	"net/http"

	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/errors"
)

// RequireRoles creates middleware that allows access only to authenticated
// identities whose role matches one of the supplied roles.
//
// Authentication and authorization are intentionally separate:
//
//   - Authentication determines WHO the caller is.
//   - Authorization determines WHETHER that caller may access a resource.
//
// This middleware assumes that authentication has already happened and that
// the authenticated identity has been stored in the request context by
// AuthenticationMiddleware.
func RequireRoles(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := AuthenticatedIdentity(r.Context())

			// No identity means authentication has not happened.
			//
			// Authorization errors are translated through the centralized
			// presentation error handler so that middleware and handlers
			// expose the same JSON error contract.
			if !ok {
				presentation_errors.WriteError(
					w,
					presentation_errors.ErrAuthenticationRequired,
				)
				return
			}

			// The caller is authenticated, but their role does not have
			// permission to access this resource.
			if !hasAllowedRole(identity.Role, allowedRoles) {
				presentation_errors.WriteError(
					w,
					presentation_errors.ErrForbidden,
				)
				return
			}

			// The caller is authenticated and authorized.
			next.ServeHTTP(w, r)
		})
	}
}

// hasAllowedRole determines whether the authenticated user's role is one
// of the roles permitted by the protected endpoint.
func hasAllowedRole(
	role string,
	allowedRoles []string,
) bool {
	for _, allowedRole := range allowedRoles {
		if role == allowedRole {
			return true
		}
	}

	return false
}
