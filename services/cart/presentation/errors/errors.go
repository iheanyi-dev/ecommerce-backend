package errors

import (
	"encoding/json"
	stderrors "errors"
	"net/http"

	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/errors"
)

type errorResponse struct {
	Error string `json:"error"`
}

var (
	ErrAuthenticationRequired    = stderrors.New("authentication required")
	ErrInvalidRequestBody        = stderrors.New("invalid request body")
	ErrMethodNotAllowed          = stderrors.New("method not allowed")
	ErrInvalidServiceCredentials = stderrors.New("invalid service credentials")
)

// WriteError maps internal errors to stable HTTP status codes and messages.
func WriteError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"

	switch {
	case stderrors.Is(err, ErrAuthenticationRequired):
		status, message = http.StatusUnauthorized, "authentication required"
	case stderrors.Is(err, ErrInvalidServiceCredentials):
		status, message = http.StatusUnauthorized, "invalid service credentials"
	case stderrors.Is(err, ErrMethodNotAllowed):
		status, message = http.StatusMethodNotAllowed, "method not allowed"
	case stderrors.Is(err, ErrInvalidRequestBody),
		stderrors.Is(err, domainerrors.ErrInvalidCustomerID),
		stderrors.Is(err, domainerrors.ErrInvalidCartID),
		stderrors.Is(err, domainerrors.ErrInvalidProductID),
		stderrors.Is(err, domainerrors.ErrInvalidStoreID),
		stderrors.Is(err, domainerrors.ErrInvalidQuantity):
		status, message = http.StatusBadRequest, "invalid request"
	case stderrors.Is(err, domainerrors.ErrProductStoreMismatch):
		status, message = http.StatusConflict, "product store mismatch"
	case stderrors.Is(err, domainerrors.ErrCartNotFound):
		status, message = http.StatusNotFound, "cart not found"
	case stderrors.Is(err, domainerrors.ErrCartItemNotFound):
		status, message = http.StatusNotFound, "cart item not found"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: message})
}
