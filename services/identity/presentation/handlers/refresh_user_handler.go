package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
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
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	var request schemas.RefreshTokenRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
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
		// Invalid, expired, revoked, or otherwise unusable refresh tokens
		// are represented by the same generic response. This prevents
		// token/session state from being unnecessarily exposed.
		if errors.Is(err, use_cases.ErrInvalidRefreshToken) {
			http.Error(
				w,
				"invalid refresh token",
				http.StatusUnauthorized,
			)
			return
		}

		// Refresh-token generation/hash failures and persistence failures
		// are internal failures. Infrastructure details must never reach
		// the client.
		if errors.Is(err, use_cases.ErrRefreshTokenGeneration) ||
			errors.Is(err, use_cases.ErrRefreshTokenHashing) ||
			errors.Is(err, use_cases.ErrRefreshTokenPersistence) ||
			errors.Is(err, use_cases.ErrTokenGeneration) {
			http.Error(
				w,
				"failed to refresh authentication",
				http.StatusInternalServerError,
			)
			return
		}

		// Any unexpected application/infrastructure failure receives the
		// same generic internal-server-error response.
		http.Error(
			w,
			"failed to refresh authentication",
			http.StatusInternalServerError,
		)
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