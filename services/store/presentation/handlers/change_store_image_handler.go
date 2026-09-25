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

// ChangeStoreImageHandler handles Store image replacement and removal.
type ChangeStoreImageHandler struct {
	service ports.ChangeStoreImageService
}

// NewChangeStoreImageHandler creates a new Store image handler.
func NewChangeStoreImageHandler(service ports.ChangeStoreImageService) *ChangeStoreImageHandler {
	return &ChangeStoreImageHandler{service: service}
}

// ServeHTTP implements http.Handler.
func (h *ChangeStoreImageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
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

	var image *dto.StoreImageInput

	file, header, err := r.FormFile("image")
	if err == nil {
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			presentation_errors.WriteError(
				w,
				presentation_errors.ErrInvalidMultipartRequest,
			)
			return
		}

		image = &dto.StoreImageInput{
			Content:     content,
			Filename:    header.Filename,
			ContentType: header.Header.Get("Content-Type"),
		}
	} else if err != http.ErrMissingFile {
		presentation_errors.WriteError(
			w,
			presentation_errors.ErrInvalidMultipartRequest,
		)
		return
	}

	result, err := h.service.Execute(
		r.Context(),
		dto.ChangeStoreImageInput{
			StoreID: storeID,
			Image:   image,
		},
	)
	if err != nil {
		presentation_errors.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, schemas.NewStoreResponse(result))
}
