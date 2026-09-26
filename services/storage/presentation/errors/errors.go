package errors

import (
	"encoding/json"
	"errors"
	"net/http"

	storage_errors "github.com/iheanyi-dev/ecommerce-backend/services/storage/infrastructure/persistence/filesystem"
)

var (
	// ErrInvalidRequest indicates that the HTTP request cannot be processed.
	ErrInvalidRequest = errors.New("invalid request")

	// ErrMethodNotAllowed indicates that the HTTP method is unsupported.
	ErrMethodNotAllowed = errors.New("method not allowed")

	// ErrInvalidServiceCredentials indicates that the caller did not provide
	// valid trusted service credentials.
	ErrInvalidServiceCredentials = errors.New("invalid service credentials")
)

type errorResponse struct {
	Error string `json:"error"`
}

// WriteError translates Storage boundary errors into safe HTTP responses.
//
// Filesystem implementation details are deliberately not exposed to callers.
func WriteError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"

	switch {
	case errors.Is(err, ErrInvalidServiceCredentials):
		status = http.StatusUnauthorized
		message = "invalid service credentials"

	case errors.Is(err, storage_errors.ErrInvalidReference),
		errors.Is(err, ErrInvalidRequest):
		status = http.StatusBadRequest
		message = "invalid request"

	case errors.Is(err, storage_errors.ErrObjectNotFound):
		status = http.StatusNotFound
		message = "object not found"

	case errors.Is(err, ErrMethodNotAllowed):
		status = http.StatusMethodNotAllowed
		message = "method not allowed"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(errorResponse{
		Error: message,
	})
}
