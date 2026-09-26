package handlers

import (
	"io"
	"net/http"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/storage/presentation/errors"
)

// GetPublicObjectHandler handles HTTP requests for retrieving objects whose
// business visibility is public.
//
// Storage itself remains internal-only. The Gateway is the public entry
// point and authenticates the request to Storage using trusted service
// credentials before this handler is reached.
//
// Object content is streamed directly from the application output to the
// HTTP response. The handler deliberately does not buffer the complete
// object in memory.
type GetPublicObjectHandler struct {
	service ports.GetPublicObjectService
}

// NewGetPublicObjectHandler creates a new public-object handler.
func NewGetPublicObjectHandler(
	service ports.GetPublicObjectService,
) *GetPublicObjectHandler {
	return &GetPublicObjectHandler{
		service: service,
	}
}

// ServeHTTP implements http.Handler.
func (h *GetPublicObjectHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		presentation_errors.WriteError(
			w,
			presentation_errors.ErrMethodNotAllowed,
		)
		return
	}

	reference := strings.TrimSpace(r.PathValue("reference"))
	if reference == "" {
		presentation_errors.WriteError(
			w,
			presentation_errors.ErrInvalidRequest,
		)
		return
	}

	result, err := h.service.Execute(
		r.Context(),
		dto.GetPublicObjectInput{
			Reference: reference,
		},
	)
	if err != nil {
		presentation_errors.WriteError(w, err)
		return
	}

	defer result.Content.Close()

	w.Header().Set("Content-Type", result.ContentType)

	// Stream the object directly to the client. Do not buffer object content
	// in memory.
	_, _ = io.Copy(w, result.Content)
}
