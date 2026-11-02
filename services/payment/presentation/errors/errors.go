// Package errors translates internal Payment service errors into safe,
// consistent HTTP responses for API clients.
package errors

import (
	"encoding/json"
	stderrors "errors"
	"net/http"

	application "github.com/iheanyi-dev/ecommerce-backend/services/payment/application"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/errors"
)

// errorResponse is the JSON structure returned to API clients.
type errorResponse struct {
	Error string `json:"error"`
}

// Presentation-layer sentinel errors represent HTTP request failures that
// do not originate from the application or domain layers.
var (
	ErrAuthenticationRequired    = stderrors.New("authentication required")
	ErrInvalidServiceCredentials = stderrors.New("invalid service credentials")
	ErrForbidden                 = stderrors.New("forbidden")
	ErrMethodNotAllowed          = stderrors.New("method not allowed")
	ErrInvalidRequestBody        = stderrors.New("invalid request body")
	ErrInvalidMultipartRequest   = stderrors.New("invalid multipart request")
)

// WriteError converts an internal error into an HTTP response.
//
// Known application, domain, and presentation errors receive explicit status
// codes and safe public messages. Unknown errors are deliberately returned
// as generic internal server errors so implementation details are not leaked.
func WriteError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"

	switch {
	// Payment and payout resources.
	case stderrors.Is(err, application.ErrPaymentNotFound):
		status, message = http.StatusNotFound, "payment not found"
	case stderrors.Is(err, application.ErrPaymentAttemptNotFound):
		status, message = http.StatusNotFound, "payment attempt not found"
	case stderrors.Is(err, application.ErrPayoutNotFound):
		status, message = http.StatusNotFound, "payout not found"
	case stderrors.Is(err, application.ErrPayoutAttemptNotFound):
		status, message = http.StatusNotFound, "payout attempt not found"

	// Duplicate resources and conflicting idempotency references.
	case stderrors.Is(err, application.ErrPaymentAlreadyExists):
		status, message = http.StatusConflict, "payment already exists"
	case stderrors.Is(err, application.ErrIdempotencyConflict):
		status, message = http.StatusConflict, "payment reference conflict"
	case stderrors.Is(err, application.ErrPayoutAlreadyExists):
		status, message = http.StatusConflict, "payout already exists"
	case stderrors.Is(err, application.ErrPayoutIdempotencyConflict):
		status, message = http.StatusConflict, "payout reference conflict"

	// Operations that cannot safely proceed in the current resource state.
	case stderrors.Is(err, application.ErrAttemptRequiresReconciliation):
		status, message = http.StatusConflict, "payment attempt requires reconciliation"
	case stderrors.Is(err, application.ErrPayoutRetryNotAllowed):
		status, message = http.StatusConflict, "payout retry not allowed"

	// Application-level request and provider failures.
	case stderrors.Is(err, application.ErrInvalidPagination):
		status, message = http.StatusBadRequest, "invalid pagination"
	case stderrors.Is(err, application.ErrInvalidProviderResponse):
		status, message = http.StatusBadGateway, "invalid payment provider response"

	// Domain validation and state-transition failures.
	case stderrors.Is(err, domainerrors.ErrInvalidPayment),
		stderrors.Is(err, domainerrors.ErrInvalidPaymentTransition),
		stderrors.Is(err, domainerrors.ErrInvalidMoney),
		stderrors.Is(err, domainerrors.ErrInvalidBusinessReference),
		stderrors.Is(err, domainerrors.ErrInvalidPaymentAttempt),
		stderrors.Is(err, domainerrors.ErrInvalidPaymentAttemptTransition),
		stderrors.Is(err, domainerrors.ErrInvalidPayout),
		stderrors.Is(err, domainerrors.ErrInvalidPayoutTransition),
		stderrors.Is(err, domainerrors.ErrInvalidPayoutAttempt),
		stderrors.Is(err, domainerrors.ErrInvalidPayoutAttemptTransition):
		status, message = http.StatusBadRequest, "invalid request"

	// Authentication and authorization failures.
	case stderrors.Is(err, ErrAuthenticationRequired):
		status, message = http.StatusUnauthorized, "authentication required"
	case stderrors.Is(err, ErrInvalidServiceCredentials):
		status, message = http.StatusUnauthorized, "invalid service credentials"
	case stderrors.Is(err, ErrForbidden):
		status, message = http.StatusForbidden, "forbidden"

	// HTTP request-shape failures.
	case stderrors.Is(err, ErrMethodNotAllowed):
		status, message = http.StatusMethodNotAllowed, "method not allowed"
	case stderrors.Is(err, ErrInvalidRequestBody),
		stderrors.Is(err, ErrInvalidMultipartRequest):
		status, message = http.StatusBadRequest, "invalid request body"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// The response is a fixed, serializable structure; encoding failures
	// should not expose internal error details to the client.
	_ = json.NewEncoder(w).Encode(errorResponse{
		Error: message,
	})
}
