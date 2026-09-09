package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/errors"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/schemas"
)

// RefreshUserHandler handles HTTP requests for access-token renewal.
//
// The handler belongs to the presentation layer and therefore knows only
// about HTTP concerns and the application-layer RefreshUserService port.
//
// It does not know how refresh tokens are generated, hashed, stored,
// validated, or rotated.
type RefreshUserHandler struct {
	refreshUserService ports.RefreshUserService
}

// NewRefreshUserHandler creates a new refresh-token HTTP handler.
//
// The application service is injected through its port so that the
// presentation layer remains independent of the concrete use case.
func NewRefreshUserHandler(
	refreshUserService ports.RefreshUserService,
) *RefreshUserHandler {
	return &RefreshUserHandler{
		refreshUserService: refreshUserService,
	}
}

// ServeHTTP implements http.Handler.
//
// Endpoint:
//
//	POST /api/v1/users/refresh
//
// The refresh endpoint intentionally remains public. It does not require
// an access token because the purpose of this endpoint is to exchange a
// valid refresh token for a new access token and replacement refresh token.
//
// The handler performs only presentation responsibilities:
//
//  1. Validate the HTTP method.
//  2. Decode the JSON request.
//  3. Convert the request into an application command.
//  4. Execute the refresh use case.
//  5. Convert the application result into an HTTP response.
func (h *RefreshUserHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		presentation_errors.WriteError(w, presentation_errors.ErrMethodNotAllowed)
		return
	}

	var request schemas.RefreshTokenRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		presentation_errors.WriteError(w, presentation_errors.ErrInvalidRequestBody)
		return
	}

	command := dto.RefreshTokenCommand{
		RefreshToken: request.RefreshToken,
	}

	result, err := h.refreshUserService.Refresh(
		r.Context(),
		command,
	)

	if err != nil {
		// Delegate application and domain error translation to the common
		// presentation error translator so every endpoint uses the same
		// HTTP status codes and JSON error response format.
		presentation_errors.WriteError(w, err)
		return
	}

	response := schemas.RefreshTokenResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(http.StatusOK)

	// The HTTP response has already been committed at this point, so an
	// encoding failure cannot be converted into a different HTTP status.
	_ = json.NewEncoder(w).Encode(response)
}
