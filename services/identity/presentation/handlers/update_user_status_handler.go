package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
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
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.PathValue("id")
	if userID == "" {
		http.Error(w, "user id is required", http.StatusBadRequest)
		return
	}

	var request schemas.UpdateUserStatusRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
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
		http.Error(w, "failed to update user status", http.StatusInternalServerError)
		return
	}

	response := schemas.NewUpdateUserStatusResponse(result)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}
