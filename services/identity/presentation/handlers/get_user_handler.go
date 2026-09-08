package handlers

import (
	"errors"
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/schemas"
)

// GetUserHandler handles administrative requests to retrieve a single user.
//
// Authentication and authorization are handled by middleware before this
// handler is reached.
//
// The handler is responsible only for HTTP concerns:
//   - validating the HTTP method
//   - obtaining the user ID from the URL path
//   - calling the application service
//   - mapping application errors to HTTP responses
//   - returning the safe user representation as JSON
type GetUserHandler struct {
	getUserService ports.GetUserService
}

// NewGetUserHandler creates a new GetUserHandler.
func NewGetUserHandler(
	getUserService ports.GetUserService,
) *GetUserHandler {
	return &GetUserHandler{
		getUserService: getUserService,
	}
}

// ServeHTTP handles:
//
//	GET /api/v1/admin/users/{id}
func (h *GetUserHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	// AuthenticationMiddleware should normally guarantee that an
	// authenticated identity exists here.
	//
	// This defensive check prevents the handler from accidentally
	// processing an unauthenticated request if the route is misconfigured.
	if _, ok := middleware.AuthenticatedIdentity(r.Context()); !ok {
		http.Error(
			w,
			"authentication required",
			http.StatusUnauthorized,
		)
		return
	}

	// The router provides the user ID through the named "id"
	// path parameter.
	userID := r.PathValue("id")

	result, err := h.getUserService.Execute(
		r.Context(),
		userID,
	)
	if err != nil {
		// ErrUserNotFound is an expected application-level condition
		// and therefore maps to HTTP 404 rather than HTTP 500.
		if errors.Is(err, use_cases.ErrUserNotFound) {
			writeJSONError(
				w,
				http.StatusNotFound,
				"user not found",
			)
			return
		}

		// Do not expose internal application or infrastructure errors
		// to the HTTP client.
		writeJSONError(
			w,
			http.StatusInternalServerError,
			"failed to get user",
		)
		return
	}

	response := schemas.NewGetUserResponse(result)

	writeJSON(
		w,
		http.StatusOK,
		response,
	)
}
