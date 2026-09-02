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
	meHandler *handlers.MeHandler,
	authenticationMiddleware *middleware.AuthenticationMiddleware,
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

	// ---------------------------------------------------------------------
	// Protected endpoints
	// ---------------------------------------------------------------------

	// Current authenticated user:
	//
	// GET /api/v1/users/me
	//
	// AuthenticationMiddleware executes before MeHandler.
	//
	// If the access token is missing, malformed, expired, invalid, or has
	// an invalid signature, the request never reaches MeHandler.
	mux.Handle(
		"/api/v1/users/me",
		authenticationMiddleware.RequireAuthentication(
			meHandler,
		),
	)

	return mux
}