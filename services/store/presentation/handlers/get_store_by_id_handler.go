package handlers

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/schemas"
)

// GetStoreByIDHandler handles HTTP requests for retrieving a Store by ID.
type GetStoreByIDHandler struct {
	service ports.GetStoreByIDService
}

// NewGetStoreByIDHandler creates a new Store-by-ID handler.
func NewGetStoreByIDHandler(service ports.GetStoreByIDService) *GetStoreByIDHandler {
	return &GetStoreByIDHandler{service: service}
}

// ServeHTTP implements http.Handler.
func (h *GetStoreByIDHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		presentation_errors.WriteError(w, presentation_errors.ErrMethodNotAllowed)
		return
	}

	storeID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presentation_errors.WriteError(w, presentation_errors.ErrInvalidRequestBody)
		return
	}

	result, err := h.service.Execute(r.Context(), storeID)
	if err != nil {
		presentation_errors.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, schemas.NewStoreResponse(result))
}
