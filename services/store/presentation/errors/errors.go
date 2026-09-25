package errors

import (
	"encoding/json"
	"errors"
	stderrors "errors"
	"net/http"

	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	domain_errors "github.com/iheanyi-dev/ecommerce-backend/services/store/domain/errors"
)

// errorResponse is the common JSON structure returned by the Store
// presentation layer whenever an HTTP request cannot be completed
// successfully.
//
// Keeping this response shape centralized ensures every Store endpoint
// exposes errors consistently, regardless of which lower layer produced
// the error.
type errorResponse struct {
	Error string `json:"error"`
}

var (
	// ErrAuthenticationRequired indicates that a protected Store endpoint
	// was reached without a trusted authenticated identity in the request
	// context.
	ErrAuthenticationRequired = stderrors.New("authentication required")

	// ErrForbidden indicates that an authenticated identity does not have
	// the role required to access the requested Store resource.
	ErrForbidden = stderrors.New("forbidden")

	// ErrMethodNotAllowed indicates that the HTTP method used for an endpoint
	// is not supported by that endpoint.
	ErrMethodNotAllowed = stderrors.New("method not allowed")

	// ErrInvalidRequestBody indicates that the HTTP request could not be
	// converted into the expected presentation-layer request.
	ErrInvalidRequestBody = stderrors.New("invalid request body")

	// ErrInvalidMultipartRequest indicates that a multipart/form-data request
	// could not be parsed or did not contain the required form structure.
	ErrInvalidMultipartRequest = stderrors.New("invalid multipart request")
)

// WriteError translates application, domain, and presentation errors into
// HTTP semantics.
//
// The application and domain layers deliberately remain unaware of HTTP.
// This translator is therefore the single Store presentation boundary where
// internal errors become HTTP status codes and public error messages.
//
// errors.Is is used so wrapped application/domain errors remain correctly
// classified.
func WriteError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"

	switch {
	case stderrors.Is(err, application_errors.ErrUnauthenticated):
		status = http.StatusUnauthorized
		message = "authentication required"

	case stderrors.Is(err, application_errors.ErrStoreAlreadyExists):
		status = http.StatusConflict
		message = "store already exists"

	case stderrors.Is(err, application_errors.ErrStoreSlugAlreadyExists):
		status = http.StatusConflict
		message = "store slug already exists"

	case stderrors.Is(err, application_errors.ErrStoreNotFound):
		status = http.StatusNotFound
		message = "store not found"

	case stderrors.Is(err, application_errors.ErrInvalidPagination):
		status = http.StatusBadRequest
		message = "invalid pagination"

	case stderrors.Is(err, domain_errors.ErrInvalidPlanType),
		stderrors.Is(err, domain_errors.ErrInvalidStoreID),
		stderrors.Is(err, domain_errors.ErrInvalidOwnerID),
		stderrors.Is(err, domain_errors.ErrInvalidStatus),
		stderrors.Is(err, domain_errors.ErrInvalidStatusTransition),
		stderrors.Is(err, domain_errors.ErrInvalidStoreName),
		stderrors.Is(err, domain_errors.ErrInvalidStoreDescription),
		stderrors.Is(err, domain_errors.ErrInvalidStoreSlug),
		stderrors.Is(err, domain_errors.ErrInvalidStoreTimestamps):
		status = http.StatusBadRequest
		message = "invalid request"

	case stderrors.Is(err, ErrAuthenticationRequired):
		status = http.StatusUnauthorized
		message = "authentication required"

	case stderrors.Is(err, ErrForbidden):
		status = http.StatusForbidden
		message = "forbidden"

	case stderrors.Is(err, ErrMethodNotAllowed):
		status = http.StatusMethodNotAllowed
		message = "method not allowed"

	case stderrors.Is(err, ErrInvalidRequestBody),
		stderrors.Is(err, ErrInvalidMultipartRequest):
		status = http.StatusBadRequest
		message = "invalid request body"

	case errors.Is(err, application_errors.ErrInvalidServiceCredentials):
		status = http.StatusUnauthorized
		message = "invalid service credentials"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// The response contains only a string, so JSON encoding should not fail.
	// If encoding somehow fails, the HTTP status has already been committed
	// and cannot safely be replaced.
	_ = json.NewEncoder(w).Encode(errorResponse{
		Error: message,
	})
}
