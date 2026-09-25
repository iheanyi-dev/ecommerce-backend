package handlers

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/schemas"
)

// RemoveStaleKnowledgeHandler handles stale knowledge removal.
type RemoveStaleKnowledgeHandler struct {
	service ports.RemoveStaleKnowledgeService
}

// NewRemoveStaleKnowledgeHandler creates a new stale-knowledge handler.
func NewRemoveStaleKnowledgeHandler(service ports.RemoveStaleKnowledgeService) *RemoveStaleKnowledgeHandler {
	return &RemoveStaleKnowledgeHandler{service: service}
}

// ServeHTTP implements http.Handler.
func (h *RemoveStaleKnowledgeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		presentation_errors.WriteError(w, presentation_errors.ErrMethodNotAllowed)
		return
	}

	storeID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presentation_errors.WriteError(w, presentation_errors.ErrInvalidRequestBody)
		return
	}

	var request schemas.RemoveStaleKnowledgeRequest

	if err := decodeJSON(r, &request); err != nil {
		presentation_errors.WriteError(w, presentation_errors.ErrInvalidRequestBody)
		return
	}

	result, err := h.service.Execute(
		r.Context(),
		dto.RemoveStaleKnowledgeInput{
			StoreID:  storeID,
			Filename: request.Filename,
		},
	)
	if err != nil {
		presentation_errors.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, schemas.RemoveStaleKnowledgeResponse{
		StoreID: result.StoreID.String(),
		Filename: result.Filename,
	})
}
