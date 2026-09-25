package handlers

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/schemas"
)

// ChangeStorePlanHandler handles Store plan changes.
type ChangeStorePlanHandler struct {
	service ports.ChangeStorePlanService
}

// NewChangeStorePlanHandler creates a new Store plan handler.
func NewChangeStorePlanHandler(service ports.ChangeStorePlanService) *ChangeStorePlanHandler {
	return &ChangeStorePlanHandler{service: service}
}

// ServeHTTP implements http.Handler.
func (h *ChangeStorePlanHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		presentation_errors.WriteError(w, presentation_errors.ErrMethodNotAllowed)
		return
	}

	storeID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presentation_errors.WriteError(w, presentation_errors.ErrInvalidRequestBody)
		return
	}

	var request schemas.ChangeStorePlanRequest

	if err := decodeJSON(r, &request); err != nil {
		presentation_errors.WriteError(w, presentation_errors.ErrInvalidRequestBody)
		return
	}

	result, err := h.service.Execute(
		r.Context(),
		dto.ChangeStorePlanInput{
			StoreID:  storeID,
			PlanType: request.PlanType,
		},
	)
	if err != nil {
		presentation_errors.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, schemas.NewStoreResponse(result))
}
