package presentation

import (
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/handlers"
)

// NewRouter creates the HTTP router for the Identity service.
//
// The router is responsible only for mapping HTTP paths and methods to
// presentation handlers. Business logic remains in the application layer.
func NewRouter(
	registerUserHandler *handlers.RegisterUserHandler,
) http.Handler {
	mux := http.NewServeMux()

	// API v1
	//
	// User registration endpoint:
	//
	// POST /api/v1/users/register
	mux.Handle(
		"/api/v1/users/register",
		registerUserHandler,
	)

	return mux
}