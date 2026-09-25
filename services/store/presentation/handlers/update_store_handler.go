package handlers

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/schemas"
)

// UpdateStoreHandler handles Store profile updates.
type UpdateStoreHandler struct {
	service ports.UpdateStoreService
}

// NewUpdateStoreHandler creates a new Store update handler.
func NewUpdateStoreHandler(service ports.UpdateStoreService) *UpdateStoreHandler {
	return &UpdateStoreHandler{service: service}
}

// ServeHTTP implements http.Handler.
func (h *UpdateStoreHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		presentation_errors.WriteError(w, presentation_errors.ErrMethodNotAllowed)
		return
	}

	storeID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presentation_errors.WriteError(w, presentation_errors.ErrInvalidRequestBody)
		return
	}

	var request schemas.UpdateStoreRequest

	if err := decodeJSON(r, &request); err != nil {
		presentation_errors.WriteError(w, presentation_errors.ErrInvalidRequestBody)
		return
	}

	result, err := h.service.Execute(
		r.Context(),
		dto.UpdateStoreInput{
			StoreID:     storeID,
			Name:        request.Name,
			Slug:        request.Slug,
			Description: request.Description,
		},
	)
	if err != nil {
		presentation_errors.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, schemas.NewStoreResponse(result))
}
