package handlers

import (
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/schemas"
)

// GetStoreByOwnerHandler handles retrieval of the authenticated owner's Store.
type GetStoreByOwnerHandler struct {
	service ports.GetStoreByOwnerService
}

// NewGetStoreByOwnerHandler creates a new Store-by-owner handler.
func NewGetStoreByOwnerHandler(service ports.GetStoreByOwnerService) *GetStoreByOwnerHandler {
	return &GetStoreByOwnerHandler{service: service}
}

// ServeHTTP implements http.Handler.
func (h *GetStoreByOwnerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		presentation_errors.WriteError(w, presentation_errors.ErrMethodNotAllowed)
		return
	}

	result, err := h.service.Execute(r.Context())
	if err != nil {
		presentation_errors.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, schemas.NewStoreResponse(*result))
}
