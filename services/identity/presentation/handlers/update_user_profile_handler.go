package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/schemas"
)

// UpdateUserProfileHandler handles authenticated self-service profile updates.
//
// The authenticated user's identity is taken exclusively from the request
// context. The request body never determines which user is updated.
type UpdateUserProfileHandler struct {
	updateUserProfileService ports.UpdateUserProfileService
}

// NewUpdateUserProfileHandler creates the profile-update HTTP handler.
func NewUpdateUserProfileHandler(
	updateUserProfileService ports.UpdateUserProfileService,
) *UpdateUserProfileHandler {
	return &UpdateUserProfileHandler{
		updateUserProfileService: updateUserProfileService,
	}
}

// ServeHTTP handles PATCH /api/v1/users/me.
func (h *UpdateUserProfileHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPatch {
		presentation_errors.WriteError(
			w,
			presentation_errors.ErrMethodNotAllowed,
		)
		return
	}
	identity, ok := middleware.AuthenticatedIdentity(r.Context())
	if !ok {
		presentation_errors.WriteError(
			w,
			presentation_errors.ErrAuthenticationRequired,
		)
		return
	}
	var request schemas.UpdateUserProfileRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid request body",
		)
		return
	}

	command := dto.UpdateUserProfileCommand{
		FullName: request.FullName,
	}

	result, err := h.updateUserProfileService.Execute(
		r.Context(),
		identity.UserID,
		command,
	)
	if err != nil {
		// Delegate application and domain error translation to the common
		// presentation error translator so every endpoint uses the same
		// HTTP status codes and JSON error response format.
		presentation_errors.WriteError(w, err)
		return
	}
	response := schemas.NewUpdateUserProfileResponse(result)

	writeJSON(w, http.StatusOK, response)
}
