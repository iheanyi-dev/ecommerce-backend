package errors

import (
	"encoding/json"
	stderrors "errors"
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/order/domain"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/infrastructure/persistence/postgres"
)

type errorResponse struct {
	Error string `json:"error"`
}

var (
	ErrAuthenticationRequired = stderrors.New("authentication required")
	ErrForbidden              = stderrors.New("forbidden")
	ErrMethodNotAllowed       = stderrors.New("method not allowed")
	ErrInvalidRequestBody     = stderrors.New("invalid request body")
)

func WriteError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"

	switch {
	case stderrors.Is(err, ErrAuthenticationRequired):
		status = http.StatusUnauthorized
		message = "authentication required"

	case stderrors.Is(err, ErrForbidden),
		stderrors.Is(err, domain.ErrUnauthorizedActor):
		status = http.StatusForbidden
		message = "forbidden"

	case stderrors.Is(err, ErrMethodNotAllowed):
		status = http.StatusMethodNotAllowed
		message = "method not allowed"

	case stderrors.Is(err, ErrInvalidRequestBody):
		status = http.StatusBadRequest
		message = "invalid request body"

	case stderrors.Is(err, postgres.ErrOrderNotFound):
		status = http.StatusNotFound
		message = "order not found"

	case stderrors.Is(err, postgres.ErrInvalidPagination):
		status = http.StatusBadRequest
		message = "invalid pagination"

	case stderrors.Is(err, domain.ErrInvalidTransition):
		status = http.StatusConflict
		message = "invalid order state transition"

	case stderrors.Is(err, domain.ErrInvalidID),
		stderrors.Is(err, domain.ErrInvalidDeliveryLocation),
		stderrors.Is(err, domain.ErrInvalidCurrency),
		stderrors.Is(err, domain.ErrNoProducts),
		stderrors.Is(err, domain.ErrInvalidProduct),
		stderrors.Is(err, domain.ErrInvalidPrice),
		stderrors.Is(err, domain.ErrInvalidQuantity),
		stderrors.Is(err, domain.ErrInvalidState),
		stderrors.Is(err, domain.ErrInvalidShippingAmount):
		status = http.StatusBadRequest
		message = "invalid request"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(errorResponse{
		Error: message,
	})
}
