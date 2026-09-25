package handlers

import (
	"net/http"
	"strconv"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/schemas"
)

// ListStoresHandler handles public Store discovery.
type ListStoresHandler struct {
	service ports.ListStoresService
}

// NewListStoresHandler creates a new Store listing handler.
func NewListStoresHandler(service ports.ListStoresService) *ListStoresHandler {
	return &ListStoresHandler{service: service}
}

// ServeHTTP implements http.Handler.
func (h *ListStoresHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		presentation_errors.WriteError(w, presentation_errors.ErrMethodNotAllowed)
		return
	}

	page := 1
	pageSize := 20

	if raw := r.URL.Query().Get("page"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			presentation_errors.WriteError(
				w,
				presentation_errors.ErrInvalidRequestBody,
			)
			return
		}

		page = value
	}

	if raw := r.URL.Query().Get("page_size"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			presentation_errors.WriteError(
				w,
				presentation_errors.ErrInvalidRequestBody,
			)
			return
		}

		pageSize = value
	}

	result, err := h.service.Execute(
		r.Context(),
		r.URL.Query().Get("query"),
		page,
		pageSize,
	)
	if err != nil {
		presentation_errors.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, schemas.NewListStoresResponse(result))
}
