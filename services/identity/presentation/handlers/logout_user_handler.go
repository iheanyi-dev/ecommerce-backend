package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/schemas"
)

// LogoutUserHandler handles HTTP requests for logging out a user.
//
// Authentication of the access token is handled by
// AuthenticationMiddleware before this handler is reached.
//
// The refresh token in the request identifies the specific
// session/device that should be revoked.
type LogoutUserHandler struct {
	logoutUserService ports.LogoutUserService
}

// NewLogoutUserHandler creates a new logout HTTP handler.
func NewLogoutUserHandler(
	logoutUserService ports.LogoutUserService,
) *LogoutUserHandler {
	return &LogoutUserHandler{
		logoutUserService: logoutUserService,
	}
}

// ServeHTTP handles:
//
//	POST /api/v1/users/logout
//
// A successful logout revokes only the refresh-token session
// supplied by the authenticated client and returns 204 No Content.
func (h *LogoutUserHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	// Logout changes server-side authentication state,
	// so it is intentionally restricted to POST requests.
	if r.Method != http.MethodPost {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	var request schemas.LogoutRequest

	// Decode the refresh token from the JSON request body.
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	// The application service hashes the refresh token,
	// finds the exact session, and revokes only that session.
	err := h.logoutUserService.Logout(
		r.Context(),
		request.RefreshToken,
	)
	if err != nil {
		// An invalid, unknown, expired, or already-revoked
		// refresh token is treated as an authentication failure.
		if errors.Is(err, use_cases.ErrInvalidRefreshToken) {
			http.Error(
				w,
				"invalid refresh token",
				http.StatusUnauthorized,
			)
			return
		}

		// Any unexpected application/infrastructure failure
		// is returned as an internal server error.
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	// Successful logout has no response body.
	w.WriteHeader(http.StatusNoContent)
}

// Compile-time assertion that the handler satisfies http.Handler.
var _ http.Handler = (*LogoutUserHandler)(nil)
