package middleware

import (
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/errors"
)

// RequireRoles creates middleware that allows access only to authenticated
// identities whose role matches one of the supplied roles.
//
// This remains the default authorization middleware and intentionally does
// not produce authorization logs or metrics. It preserves the existing
// behavior for callers that do not require authorization observability.
func RequireRoles(allowedRoles ...string) func(http.Handler) http.Handler {
	return RequireRolesWithObservability(
		nil,
		nil,
		allowedRoles...,
	)
}

// RequireRolesWithLogger creates authorization middleware with optional
// structured logging.
//
// Authorization successes are intentionally not logged because they provide
// little operational value and would create unnecessary log volume.
//
// Authorization denials are logged because they provide useful security and
// audit information that the generic request observability middleware cannot
// determine, particularly the role required by the protected resource.
func RequireRolesWithLogger(
	logger ports.Logger,
	allowedRoles ...string,
) func(http.Handler) http.Handler {
	return RequireRolesWithObservability(
		logger,
		nil,
		allowedRoles...,
	)
}

// RequireRolesWithObservability creates authorization middleware with optional
// structured logging and metrics.
//
// Authorization denials increment a low-cardinality metric:
//
//	auth.authorization{result="denied"}
//
// No user ID, role, route, or required role is included in the metric labels.
// This prevents high-cardinality metric data while still allowing operators
// to monitor authorization failures.
func RequireRolesWithObservability(
	logger ports.Logger,
	metrics ports.Metrics,
	allowedRoles ...string,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := AuthenticatedIdentity(r.Context())

			// No identity means authentication has not happened.
			//
			// This is deliberately not logged or counted as an authorization
			// denial here. The request-level observability middleware records
			// the resulting 401 response, avoiding duplicate authentication
			// related observability events.
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

				recordAuthorizationDenial(
					r,
					metrics,
				)

				logAuthorizationDenial(
					r,
					identity,
					allowedRoles,
					logger,
				)

				return
			}

			// The caller is authenticated and authorized.
			//
			// Successful authorization is intentionally not logged or
			// counted because it provides little operational value.
			next.ServeHTTP(w, r)
		})
	}
}

// recordAuthorizationDenial records a low-cardinality authorization failure
// metric.
//
// Metrics are best-effort. A metrics infrastructure failure must never change
// the authorization response.
func recordAuthorizationDenial(
	r *http.Request,
	metrics ports.Metrics,
) {
	if metrics == nil {
		return
	}

	_ = metrics.Increment(
		r.Context(),
		ports.Metric{
			Name:  "auth.authorization",
			Value: 1,
			Labels: map[string]string{
				"result": "denied",
			},
		},
	)
}

// logAuthorizationDenial records a structured security event for a forbidden
// request.
//
// Logging is best-effort. A failure in the logging infrastructure must never
// change the HTTP response produced by the authorization middleware.
func logAuthorizationDenial(
	r *http.Request,
	identity ports.AuthenticatedIdentity,
	allowedRoles []string,
	logger ports.Logger,
) {
	if logger == nil {
		return
	}

	route := r.Pattern
	if route == "" {
		route = r.URL.Path
	}

	// RequiredRole represents the authorization requirement that caused the
	// denial. For the current role-based middleware, recording the permitted
	// roles as a comma-separated value gives us the complete requirement
	// without introducing another field into the logging contract.
	requiredRole := ""

	for index, role := range allowedRoles {
		if index > 0 {
			requiredRole += ","
		}

		requiredRole += role
	}

	event := ports.LogEvent{
		Event:           "http.authorization.denied",
		Operation:       route,
		UserID:          identity.UserID,
		Role:            identity.Role,
		HTTPMethod:      r.Method,
		Route:           route,
		StatusCode:      http.StatusForbidden,
		FailureCategory: "client_error",
		RequiredRole:    requiredRole,
	}

	// Never allow observability failures to affect authorization behavior.
	_ = logger.Log(r.Context(), event)
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
