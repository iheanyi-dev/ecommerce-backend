package presentation

import (
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
)

// NewRouter creates the HTTP router for the Identity service.
//
// The router is responsible only for mapping HTTP paths and methods to
// presentation handlers.
//
// Authentication and authorization remain separate middleware concerns.
func NewRouter(
	registerUserHandler *handlers.RegisterUserHandler,
	loginUserHandler *handlers.LoginUserHandler,
	refreshUserHandler *handlers.RefreshUserHandler,
	meHandler *handlers.MeHandler,
	authenticationMiddleware *middleware.AuthenticationMiddleware,
	logoutUserHandler *handlers.LogoutUserHandler,
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
		"/api/v1/users/me",
		authenticationMiddleware.RequireAuthentication(
			middleware.RequireRoles(
				"admin",
				"vendor",
				"user",
			)(
				meHandler,
			),
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

	return mux
}
