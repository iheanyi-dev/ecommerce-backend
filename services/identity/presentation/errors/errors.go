package errors

import (
	"encoding/json"
	stderrors "errors"
	"net/http"

	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	domain_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/errors"
)

// errorResponse is the common JSON structure returned by the presentation
// layer whenever an HTTP request cannot be completed successfully.
//
// Keeping the response structure centralized ensures that all handlers and
// middleware expose errors to API clients in a consistent format.
type errorResponse struct {
	Error string `json:"error"`
}

var (
	// ErrAuthenticationRequired indicates that authorization was attempted
	// without an authenticated identity being present in the request context.
	ErrAuthenticationRequired = stderrors.New("authentication required")

	// ErrForbidden indicates that an authenticated identity does not have
	// the role required to access the protected resource.
	ErrForbidden = stderrors.New("forbidden")
	// ErrMethodNotAllowed indicates that the HTTP method used for an endpoint
	// is not supported by that endpoint.
	ErrMethodNotAllowed = stderrors.New("method not allowed")
	// ErrUserIDRequired indicates that an endpoint requiring a user ID
	// did not receive one in the URL path.
	ErrUserIDRequired = stderrors.New("user id is required")

	// ErrInvalidRequestBody indicates that the HTTP request body could not
	// be decoded into the expected request schema.
	ErrInvalidRequestBody = stderrors.New("invalid request body")
)

// WriteError translates an application or domain error into an appropriate
// HTTP response.
//
// The presentation layer is responsible for converting internal errors into
// HTTP semantics. Lower layers should never need to know about HTTP status
// codes or response formats.
//
// errors.Is is intentionally used throughout this function so that wrapped
// errors are handled correctly.
func WriteError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"

	switch {
	// Authentication failures must not reveal whether a specific user
	// exists or which part of authentication failed.
	case stderrors.Is(err, application_errors.ErrInvalidCredentials):
		status = http.StatusUnauthorized
		message = "invalid credentials"

	// Inactive accounts cannot authenticate successfully.
	case stderrors.Is(err, application_errors.ErrAccountNotActive):
		status = http.StatusUnauthorized
		message = "account is not active"

	// A duplicate email represents a conflict with an existing resource.
	case stderrors.Is(err, application_errors.ErrEmailAlreadyExists):
		status = http.StatusConflict
		message = "email already exists"

	// A requested user does not exist.
	case stderrors.Is(err, application_errors.ErrUserNotFound):
		status = http.StatusNotFound
		message = "user not found"

	// Invalid refresh tokens are treated as authentication failures.
	case stderrors.Is(err, application_errors.ErrInvalidRefreshToken):
		status = http.StatusUnauthorized
		message = "invalid refresh token"

	// Domain validation failures are client-side input errors.
	case stderrors.Is(err, domain_errors.ErrInvalidEmail),
		stderrors.Is(err, domain_errors.ErrInvalidFullName),
		stderrors.Is(err, domain_errors.ErrInvalidPasswordHash),
		stderrors.Is(err, domain_errors.ErrInvalidRole),
		stderrors.Is(err, domain_errors.ErrInvalidStatus),
		stderrors.Is(err, domain_errors.ErrInvalidStatusTransition),
		stderrors.Is(err, domain_errors.ErrInvalidUserID),
		stderrors.Is(err, domain_errors.ErrUserAlreadyVendor),
		stderrors.Is(err, domain_errors.ErrInvalidVendorPromotion):
		status = http.StatusBadRequest
		message = "invalid request"

	case stderrors.Is(err, application_errors.ErrInvalidAccessToken):
		status = http.StatusUnauthorized
		message = "invalid or expired access token"

	case stderrors.Is(err, ErrAuthenticationRequired):
		status = http.StatusUnauthorized
		message = "authentication required"

	case stderrors.Is(err, ErrForbidden):
		status = http.StatusForbidden
		message = "forbidden"

		// The requested HTTP method is not supported by the endpoint.
	case stderrors.Is(err, ErrMethodNotAllowed):
		status = http.StatusMethodNotAllowed
		message = "method not allowed"

		// A required user ID was missing from the request path.
	case stderrors.Is(err, ErrUserIDRequired):
		status = http.StatusBadRequest
		message = "user id is required"

	// The request body could not be decoded into the expected schema.
	case stderrors.Is(err, ErrInvalidRequestBody):
		status = http.StatusBadRequest
		message = "invalid request body"
	}

	// Always return JSON so API clients can consume errors consistently.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Encoding the response should not normally fail because the response
	// contains only a string. If it somehow does, the status has already been
	// written, so there is no safe way to replace the response at this point.
	_ = json.NewEncoder(w).Encode(errorResponse{
		Error: message,
	})
}
