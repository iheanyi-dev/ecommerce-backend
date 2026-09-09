package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/schemas"
)

// UpdateUserStatusHandler handles administrative account-status updates.
//
// Authorization is intentionally handled by the router/middleware. This
// handler is responsible only for HTTP concerns and invoking the application
// service.
type UpdateUserStatusHandler struct {
	service ports.UpdateUserStatusService
}

// NewUpdateUserStatusHandler constructs the administrative status handler.
func NewUpdateUserStatusHandler(
	service ports.UpdateUserStatusService,
) *UpdateUserStatusHandler {
	return &UpdateUserStatusHandler{
		service: service,
	}
}

// ServeHTTP handles PATCH /api/v1/admin/users/{id}/status.
func (h *UpdateUserStatusHandler) ServeHTTP(
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

	userID := r.PathValue("id")
	if userID == "" {
		presentation_errors.WriteError(
			w,
			presentation_errors.ErrUserIDRequired,
		)
		return
	}
	var request schemas.UpdateUserStatusRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		presentation_errors.WriteError(
			w,
			presentation_errors.ErrInvalidRequestBody,
		)
		return
	}

	command := dto.UpdateUserStatusCommand{
		Status: request.Status,
	}
	result, err := h.service.Execute(
		r.Context(),
		userID,
		command,
	)
	if err != nil {
		// Delegate application and domain error translation to the common
		// presentation error translator so every endpoint uses the same
		// HTTP status codes and JSON error response format.
		presentation_errors.WriteError(w, err)
		return
	}

	response := schemas.NewUpdateUserStatusResponse(result)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}
