package handlers

import (
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/schemas"
)

// GetStoreBySlugHandler handles HTTP requests for retrieving a Store by slug.
type GetStoreBySlugHandler struct {
	service ports.GetStoreBySlugService
}

// NewGetStoreBySlugHandler creates a new Store-by-slug handler.
func NewGetStoreBySlugHandler(service ports.GetStoreBySlugService) *GetStoreBySlugHandler {
	return &GetStoreBySlugHandler{service: service}
}

// ServeHTTP implements http.Handler.
func (h *GetStoreBySlugHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		presentation_errors.WriteError(w, presentation_errors.ErrMethodNotAllowed)
		return
	}

	slug := r.PathValue("slug")
	if slug == "" {
		presentation_errors.WriteError(w, presentation_errors.ErrInvalidRequestBody)
		return
	}

	result, err := h.service.Execute(r.Context(), slug)
	if err != nil {
		presentation_errors.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, schemas.NewStoreResponse(*result))
}
