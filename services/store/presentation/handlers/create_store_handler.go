package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/schemas"
)

// CreateStoreHandler handles HTTP requests for Store creation.
//
// The handler is responsible only for presentation concerns:
//   - validating the HTTP method
//   - decoding multipart/form-data
//   - translating uploaded image data into the application DTO
//   - invoking the application service
//   - translating application errors into HTTP responses
//   - converting the application result into the public response schema
//
// Authentication is intentionally not performed here. The trusted
// authentication middleware establishes the authenticated identity in the
// request context before the request reaches this handler.
type CreateStoreHandler struct {
	createStoreService ports.CreateStoreService
}

// NewCreateStoreHandler creates a new Store creation HTTP handler.
func NewCreateStoreHandler(
	createStoreService ports.CreateStoreService,
) *CreateStoreHandler {
	return &CreateStoreHandler{
		createStoreService: createStoreService,
	}
}

// ServeHTTP implements http.Handler.
func (h *CreateStoreHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	// Store creation uses multipart/form-data because the request may contain
	// both Store fields and an optional image upload.
	if r.Method != http.MethodPost {
		presentation_errors.WriteError(
			w,
			presentation_errors.ErrMethodNotAllowed,
		)
		return
	}

	// ParseMultipartForm keeps the HTTP-specific multipart handling inside the
	// presentation layer. The application layer receives only plain Go data.
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		presentation_errors.WriteError(
			w,
			presentation_errors.ErrInvalidMultipartRequest,
		)
		return
	}

	input := dto.CreateStoreInput{
		Name:        r.FormValue("name"),
		Slug:        r.FormValue("slug"),
		Description: r.FormValue("description"),
	}

	// The image is optional. When supplied, copy its bytes into an
	// application-level DTO so the multipart file handle never crosses the
	// presentation boundary.
	file, header, err := r.FormFile("image")
	if err == nil {
		defer file.Close()

		content, readErr := io.ReadAll(file)
		if readErr != nil {
			presentation_errors.WriteError(
				w,
				presentation_errors.ErrInvalidMultipartRequest,
			)
			return
		}

		input.Image = &dto.StoreImageInput{
			Content:     content,
			Filename:    header.Filename,
			ContentType: header.Header.Get("Content-Type"),
		}
	} else if err != http.ErrMissingFile {
		// A missing image is valid, but an actual multipart file-processing
		// error is not. Do not silently convert malformed uploads into a
		// request without an image.
		presentation_errors.WriteError(
			w,
			presentation_errors.ErrInvalidMultipartRequest,
		)
		return
	}

	// Execute the application workflow. The application layer is responsible
	// for authentication authorization, domain construction, persistence,
	// Identity synchronization, and asynchronous dispatches.
	result, err := h.createStoreService.Execute(
		r.Context(),
		input,
	)
	if err != nil {
		presentation_errors.WriteError(w, err)
		return
	}

	// Convert domain/application types into the stable HTTP API contract.
	response := schemas.NewCreateStoreResponse(result)

	writeJSON(
		w,
		http.StatusCreated,
		response,
	)
}

// writeJSON writes a successful JSON response.
func writeJSON(
	w http.ResponseWriter,
	status int,
	data any,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// At this point the response has already started. There is no useful
	// HTTP-level recovery action if JSON encoding fails, so the error is
	// intentionally ignored.
	_ = json.NewEncoder(w).Encode(data)
}
