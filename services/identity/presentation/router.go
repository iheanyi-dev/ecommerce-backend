package presentation

import (
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
)

// NewRouter creates the HTTP router for the Identity service.
//
// The router is responsible only for mapping HTTP paths and methods to
// presentation handlers and composing the presentation middleware chain.
//
// Authentication and authorization remain separate middleware concerns.
//
// The logger is passed explicitly so authorization-denial observability does
// not depend on hidden or package-level state.
//
// Request IDs are applied around the complete router so every Identity
// request receives a correlation identifier, including requests rejected by
// service authentication.
//
// Request-level observability is applied around the completed router so that
// it can record the final HTTP status code produced by handlers and
// middleware.
func NewRouter(
	registerUserHandler *handlers.RegisterUserHandler,
	loginUserHandler *handlers.LoginUserHandler,
	refreshUserHandler *handlers.RefreshUserHandler,
	meHandler *handlers.MeHandler,
	updateUserProfileHandler *handlers.UpdateUserProfileHandler,
	authenticationMiddleware *middleware.AuthenticationMiddleware,
	logoutUserHandler *handlers.LogoutUserHandler,
	listUsersHandler *handlers.ListUsersHandler,
	getUserHandler *handlers.GetUserHandler,
	updateUserStatusHandler *handlers.UpdateUserStatusHandler,
	logger ports.Logger,
	requestObservabilityMiddleware *middleware.RequestObservabilityMiddleware,
	tracingMiddleware *middleware.TracingMiddleware,
	serviceAuthenticationMiddleware *middleware.ServiceAuthenticationMiddleware,
	requestIDMiddleware *middleware.RequestIDMiddleware,
) http.Handler {
	mux := http.NewServeMux()

	// ---------------------------------------------------------------------
	// Public endpoints
	// ---------------------------------------------------------------------

	// User registration:
	//
	// POST /api/v1/users/register
	mux.Handle(
		"/api/v1/users/register",
		registerUserHandler,
	)

	// User authentication:
	//
	// POST /api/v1/users/login
	//
	// This endpoint must remain public because users need to authenticate
	// before they can obtain an access token.
	mux.Handle(
		"/api/v1/users/login",
		loginUserHandler,
	)

	// Refresh authentication tokens:
	//
	// POST /api/v1/users/refresh
	//
	// This endpoint intentionally remains public because the caller does
	// not present an access token. Instead, the refresh token itself is
	// supplied in the request body and validated by the application layer.
	mux.Handle(
		"/api/v1/users/refresh",
		refreshUserHandler,
	)

	// ---------------------------------------------------------------------
	// Protected endpoints
	// ---------------------------------------------------------------------

	// Current authenticated user:
	//
	// GET /api/v1/users/me
	//
	// The middleware chain is intentionally ordered:
	//
	//      Authentication → Authorization → Handler
	//
	// Authentication first establishes WHO the caller is.
	// Authorization then determines WHETHER the caller's role is allowed.
	//
	// All valid application roles can access their own profile.
	mux.Handle(
		"GET /api/v1/users/me",
		authenticationMiddleware.RequireAuthentication(
			middleware.RequireRolesWithLogger(
				logger,
				"admin",
				"vendor",
				"user",
			)(meHandler),
		),
	)

	mux.Handle(
		"PATCH /api/v1/users/me",
		authenticationMiddleware.RequireAuthentication(
			middleware.RequireRolesWithLogger(
				logger,
				"admin",
				"vendor",
				"user",
			)(updateUserProfileHandler),
		),
	)

	// Logout requires a valid access token, but does not require
	// a specific role because every authenticated user can log out.
	mux.Handle(
		"/api/v1/users/logout",
		authenticationMiddleware.RequireAuthentication(
			logoutUserHandler,
		),
	)

	// ---------------------------------------------------------------------
	// Administrative endpoints
	// ---------------------------------------------------------------------

	// List users:
	//
	// GET /api/v1/admin/users
	//
	// Only administrators may access this endpoint.
	//
	// The middleware chain is intentionally:
	//
	//      Authentication → Authorization → Handler
	//
	// Authentication establishes the caller's identity.
	// Authorization then restricts access to the admin role.
	mux.Handle(
		"GET /api/v1/admin/users",
		authenticationMiddleware.RequireAuthentication(
			middleware.RequireRolesWithLogger(
				logger,
				"admin",
			)(listUsersHandler),
		),
	)

	// Get a single user:
	//
	// GET /api/v1/admin/users/{id}
	//
	// Only administrators may access this endpoint.
	//
	// Authentication establishes the caller's identity.
	// Authorization then restricts access to the admin role.
	mux.Handle(
		"GET /api/v1/admin/users/{id}",
		authenticationMiddleware.RequireAuthentication(
			middleware.RequireRolesWithLogger(
				logger,
				"admin",
			)(getUserHandler),
		),
	)

	// Update a user's account status:
	//
	// PATCH /api/v1/admin/users/{id}/status
	//
	// Only administrators may access this endpoint.
	//
	// Authentication establishes the caller's identity.
	// Authorization then restricts access to the admin role.
	//
	// The handler itself only receives the target user ID and requested
	// account status. Role, email, password, and profile fields are not
	// part of this administrative operation.
	mux.Handle(
		"PATCH /api/v1/admin/users/{id}/status",
		authenticationMiddleware.RequireAuthentication(
			middleware.RequireRolesWithLogger(
				logger,
				"admin",
			)(updateUserStatusHandler),
		),
	)

	// The middleware order around the complete router is intentionally:
	//
	//      Request ID
	//          ↓
	//      Request Observability
	//          ↓
	//      Tracing
	//          ↓
	//      Service Authentication
	//          ↓
	//      Router / endpoint middleware
	//
	// Request ID is outermost so every request, including requests rejected
	// by service authentication, receives a correlation identifier.
	var handler http.Handler = mux

	// Service authentication is the Identity service boundary.
	//
	// This ensures every request reaching Identity has first been
	// authenticated as a trusted internal service. User authentication
	// remains a separate concern inside protected routes.
	if serviceAuthenticationMiddleware != nil {
		handler = serviceAuthenticationMiddleware.RequireServiceAuthentication(
			handler,
		)
	}

	if tracingMiddleware != nil {
		handler = tracingMiddleware.Middleware(handler)
	}

	if requestObservabilityMiddleware != nil {
		handler = requestObservabilityMiddleware.Middleware(handler)
	}

	if requestIDMiddleware != nil {
		handler = requestIDMiddleware.Middleware(handler)
	}

	return handler
}
