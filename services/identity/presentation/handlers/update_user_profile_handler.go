package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
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
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	identity, ok := middleware.AuthenticatedIdentity(r.Context())
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
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
		switch {
		case errors.Is(err, use_cases.ErrUserNotFound):
			writeJSONError(
				w,
				http.StatusNotFound,
				"user not found",
			)

		case errors.Is(err, user.ErrInvalidFullName):
			writeJSONError(
				w,
				http.StatusBadRequest,
				"invalid full name",
			)

		default:
			writeJSONError(
				w,
				http.StatusInternalServerError,
				"failed to update user profile",
			)
		}

		return
	}

	response := schemas.NewUpdateUserProfileResponse(result)

	writeJSON(w, http.StatusOK, response)
}
