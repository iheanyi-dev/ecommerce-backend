package handlers

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/schemas"
)

// ChangeStoreStatusHandler handles trusted internal Store status changes.
//
// Status changes are controlled by subscription/billing workflows rather than
// ordinary Store-owner requests. Authentication of the trusted internal
// caller belongs to the service-boundary middleware.
type ChangeStoreStatusHandler struct {
	service ports.ChangeStoreStatusService
}

// NewChangeStoreStatusHandler creates a new Store status handler.
func NewChangeStoreStatusHandler(service ports.ChangeStoreStatusService) *ChangeStoreStatusHandler {
	return &ChangeStoreStatusHandler{service: service}
}

// ServeHTTP implements http.Handler.
func (h *ChangeStoreStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		presentation_errors.WriteError(w, presentation_errors.ErrMethodNotAllowed)
		return
	}

	storeID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presentation_errors.WriteError(w, presentation_errors.ErrInvalidRequestBody)
		return
	}

	var request schemas.ChangeStoreStatusRequest

	if err := decodeJSON(r, &request); err != nil {
		presentation_errors.WriteError(w, presentation_errors.ErrInvalidRequestBody)
		return
	}

	result, err := h.service.Execute(
		r.Context(),
		dto.ChangeStoreStatusInput{
			StoreID: storeID,
			Status:  request.Status,
		},
	)
	if err != nil {
		presentation_errors.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, schemas.NewStoreResponse(result))
}
