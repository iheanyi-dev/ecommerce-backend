package handlers

import (
	"io"
	"net/http"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/schemas"
)

// AddStoreKnowledgeHandler handles Store knowledge-file uploads.
type AddStoreKnowledgeHandler struct {
	service ports.AddStoreKnowledgeService
}

// NewAddStoreKnowledgeHandler creates a new Store knowledge handler.
func NewAddStoreKnowledgeHandler(service ports.AddStoreKnowledgeService) *AddStoreKnowledgeHandler {
	return &AddStoreKnowledgeHandler{service: service}
}

// ServeHTTP implements http.Handler.
func (h *AddStoreKnowledgeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		presentation_errors.WriteError(w, presentation_errors.ErrMethodNotAllowed)
		return
	}

	storeID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presentation_errors.WriteError(w, presentation_errors.ErrInvalidRequestBody)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		presentation_errors.WriteError(w, presentation_errors.ErrInvalidMultipartRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		presentation_errors.WriteError(w, presentation_errors.ErrInvalidMultipartRequest)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		presentation_errors.WriteError(w, presentation_errors.ErrInvalidMultipartRequest)
		return
	}

	result, err := h.service.Execute(
		r.Context(),
		dto.AddStoreKnowledgeInput{
			StoreID: storeID,
			File: dto.StoreKnowledgeFileInput{
				Content:     content,
				Filename:    header.Filename,
				ContentType: header.Header.Get("Content-Type"),
			},
		},
	)
	if err != nil {
		presentation_errors.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, schemas.AddStoreKnowledgeResponse{
		StoreID: result.StoreID.String(),
	})
}
