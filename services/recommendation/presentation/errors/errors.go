package errors

import (
	"encoding/json"
	"errors"
	"net/http"
)

// ErrorResponse represents an HTTP error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// WriteError writes a JSON error response.
func WriteError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error: message,
	})
}

// WriteApplicationError writes an application error response.
func WriteApplicationError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	var clientError interface {
		HTTPStatus() int
	}

	if errors.As(err, &clientError) {
		WriteError(w, clientError.HTTPStatus(), err.Error())
		return
	}

	WriteError(w, http.StatusInternalServerError, "internal server error")
}
