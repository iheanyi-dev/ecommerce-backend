package middleware

import "net/http"

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
			if !ok {
				http.Error(
					w,
					"authentication required",
					http.StatusUnauthorized,
				)
				return
			}

			// The caller is authenticated, but their role does not have
			// permission to access this resource.
			if !hasAllowedRole(identity.Role, allowedRoles) {
				http.Error(
					w,
					"forbidden",
					http.StatusForbidden,
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
