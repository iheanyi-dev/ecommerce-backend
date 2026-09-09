package errors_test

import (
	stderrors "errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	domain_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/errors"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/errors"
)

// TestWriteError_InvalidCredentials verifies that invalid authentication
// credentials are translated into an HTTP 401 Unauthorized response.
func TestWriteError_InvalidCredentials(t *testing.T) {
	recorder := httptest.NewRecorder()

	presentation_errors.WriteError(
		recorder,
		application_errors.ErrInvalidCredentials,
	)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			recorder.Code,
		)
	}

	expectedBody := `{"error":"invalid credentials"}` + "\n"

	if recorder.Body.String() != expectedBody {
		t.Fatalf(
			"expected body %q, got %q",
			expectedBody,
			recorder.Body.String(),
		)
	}
}

// TestWriteError_AccountNotActive verifies that an inactive account is
// translated into an HTTP 401 Unauthorized response.
func TestWriteError_AccountNotActive(t *testing.T) {
	recorder := httptest.NewRecorder()

	presentation_errors.WriteError(
		recorder,
		application_errors.ErrAccountNotActive,
	)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			recorder.Code,
		)
	}
}

// TestWriteError_EmailAlreadyExists verifies that a duplicate email is
// translated into an HTTP 409 Conflict response.
func TestWriteError_EmailAlreadyExists(t *testing.T) {
	recorder := httptest.NewRecorder()

	presentation_errors.WriteError(
		recorder,
		application_errors.ErrEmailAlreadyExists,
	)

	if recorder.Code != http.StatusConflict {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusConflict,
			recorder.Code,
		)
	}
}

// TestWriteError_UserNotFound verifies that a missing user is translated
// into an HTTP 404 Not Found response.
func TestWriteError_UserNotFound(t *testing.T) {
	recorder := httptest.NewRecorder()

	presentation_errors.WriteError(
		recorder,
		application_errors.ErrUserNotFound,
	)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			recorder.Code,
		)
	}
}

// TestWriteError_DomainValidationError verifies that a domain validation
// failure is translated into an HTTP 400 Bad Request response.
func TestWriteError_DomainValidationError(t *testing.T) {
	recorder := httptest.NewRecorder()

	presentation_errors.WriteError(
		recorder,
		domain_errors.ErrInvalidEmail,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

// TestWriteError_AllDomainValidationErrors verifies that every domain
// validation error is exposed through the same HTTP 400 contract.
//
// Domain validation remains a domain concern; the presentation layer only
// decides that these failures represent a bad client request.
func TestWriteError_AllDomainValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "invalid email",
			err:  domain_errors.ErrInvalidEmail,
		},
		{
			name: "invalid full name",
			err:  domain_errors.ErrInvalidFullName,
		},
		{
			name: "invalid password hash",
			err:  domain_errors.ErrInvalidPasswordHash,
		},
		{
			name: "invalid role",
			err:  domain_errors.ErrInvalidRole,
		},
		{
			name: "invalid status",
			err:  domain_errors.ErrInvalidStatus,
		},
		{
			name: "invalid status transition",
			err:  domain_errors.ErrInvalidStatusTransition,
		},
		{
			name: "invalid user ID",
			err:  domain_errors.ErrInvalidUserID,
		},
		{
			name: "user already vendor",
			err:  domain_errors.ErrUserAlreadyVendor,
		},
		{
			name: "invalid vendor promotion",
			err:  domain_errors.ErrInvalidVendorPromotion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()

			presentation_errors.WriteError(
				recorder,
				tt.err,
			)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf(
					"expected status %d, got %d",
					http.StatusBadRequest,
					recorder.Code,
				)
			}

			if recorder.Body.String() != `{"error":"invalid request"}`+"\n" {
				t.Fatalf(
					"expected body %q, got %q",
					`{"error":"invalid request"}`+"\n",
					recorder.Body.String(),
				)
			}

			if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
				t.Fatalf(
					"expected Content-Type %q, got %q",
					"application/json",
					contentType,
				)
			}
		})
	}
}

// TestWriteError_ApplicationErrorContract verifies the complete HTTP
// contract for application-layer errors handled by the centralized
// presentation translator.
func TestWriteError_ApplicationErrorContract(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "invalid credentials",
			err:            application_errors.ErrInvalidCredentials,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"invalid credentials"}` + "\n",
		},
		{
			name:           "account not active",
			err:            application_errors.ErrAccountNotActive,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"account is not active"}` + "\n",
		},
		{
			name:           "email already exists",
			err:            application_errors.ErrEmailAlreadyExists,
			expectedStatus: http.StatusConflict,
			expectedBody:   `{"error":"email already exists"}` + "\n",
		},
		{
			name:           "user not found",
			err:            application_errors.ErrUserNotFound,
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":"user not found"}` + "\n",
		},
		{
			name:           "invalid refresh token",
			err:            application_errors.ErrInvalidRefreshToken,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"invalid refresh token"}` + "\n",
		},
		{
			name:           "invalid access token",
			err:            application_errors.ErrInvalidAccessToken,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"invalid or expired access token"}` + "\n",
		},
		{
			name:           "token generation",
			err:            application_errors.ErrTokenGeneration,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"internal server error"}` + "\n",
		},
		{
			name:           "refresh token generation",
			err:            application_errors.ErrRefreshTokenGeneration,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"internal server error"}` + "\n",
		},
		{
			name:           "refresh token hashing",
			err:            application_errors.ErrRefreshTokenHashing,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"internal server error"}` + "\n",
		},
		{
			name:           "refresh token persistence",
			err:            application_errors.ErrRefreshTokenPersistence,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"internal server error"}` + "\n",
		},
		{
			name:           "refresh token revocation",
			err:            application_errors.ErrRefreshTokenRevocation,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"internal server error"}` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()

			presentation_errors.WriteError(
				recorder,
				tt.err,
			)

			if recorder.Code != tt.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d",
					tt.expectedStatus,
					recorder.Code,
				)
			}

			if recorder.Body.String() != tt.expectedBody {
				t.Fatalf(
					"expected body %q, got %q",
					tt.expectedBody,
					recorder.Body.String(),
				)
			}

			if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
				t.Fatalf(
					"expected Content-Type %q, got %q",
					"application/json",
					contentType,
				)
			}
		})
	}
}

// TestWriteError_PresentationErrorContract verifies the errors that are
// defined directly by the presentation layer.
func TestWriteError_PresentationErrorContract(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "authentication required",
			err:            presentation_errors.ErrAuthenticationRequired,
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"authentication required"}` + "\n",
		},
		{
			name:           "forbidden",
			err:            presentation_errors.ErrForbidden,
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"error":"forbidden"}` + "\n",
		},
		{
			name:           "method not allowed",
			err:            presentation_errors.ErrMethodNotAllowed,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   `{"error":"method not allowed"}` + "\n",
		},
		{
			name:           "user ID required",
			err:            presentation_errors.ErrUserIDRequired,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"user id is required"}` + "\n",
		},
		{
			name:           "invalid request body",
			err:            presentation_errors.ErrInvalidRequestBody,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"invalid request body"}` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()

			presentation_errors.WriteError(
				recorder,
				tt.err,
			)

			if recorder.Code != tt.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d",
					tt.expectedStatus,
					recorder.Code,
				)
			}

			if recorder.Body.String() != tt.expectedBody {
				t.Fatalf(
					"expected body %q, got %q",
					tt.expectedBody,
					recorder.Body.String(),
				)
			}

			if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
				t.Fatalf(
					"expected Content-Type %q, got %q",
					"application/json",
					contentType,
				)
			}
		})
	}
}

// TestWriteError_WrappedError verifies that the translator uses errors.Is
// semantics and can therefore recognize application errors wrapped by
// another error.
func TestWriteError_WrappedError(t *testing.T) {
	recorder := httptest.NewRecorder()

	wrappedErr := fmt.Errorf(
		"operation failed: %w",
		application_errors.ErrUserNotFound,
	)

	presentation_errors.WriteError(
		recorder,
		wrappedErr,
	)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d for wrapped error, got %d",
			http.StatusNotFound,
			recorder.Code,
		)
	}
}

// TestWriteError_WrappedAuthorizationError verifies that authorization
// errors remain recognizable when wrapped by another error.
//
// This preserves the errors.Is contract used by the centralized translator.
func TestWriteError_WrappedAuthorizationError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "wrapped authentication required",
			err: fmt.Errorf(
				"middleware failure: %w",
				presentation_errors.ErrAuthenticationRequired,
			),
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"authentication required"}` + "\n",
		},
		{
			name: "wrapped forbidden",
			err: fmt.Errorf(
				"middleware failure: %w",
				presentation_errors.ErrForbidden,
			),
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"error":"forbidden"}` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()

			presentation_errors.WriteError(
				recorder,
				tt.err,
			)

			if recorder.Code != tt.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d",
					tt.expectedStatus,
					recorder.Code,
				)
			}

			if recorder.Body.String() != tt.expectedBody {
				t.Fatalf(
					"expected body %q, got %q",
					tt.expectedBody,
					recorder.Body.String(),
				)
			}
		})
	}
}

// TestWriteError_UnexpectedError verifies that unexpected internal errors
// are translated into HTTP 500 Internal Server Error without exposing
// internal implementation details to the client.
func TestWriteError_UnexpectedError(t *testing.T) {
	recorder := httptest.NewRecorder()

	internalErr := stderrors.New("database password leaked")

	presentation_errors.WriteError(
		recorder,
		internalErr,
	)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			recorder.Code,
		)
	}

	expectedBody := `{"error":"internal server error"}` + "\n"

	if recorder.Body.String() != expectedBody {
		t.Fatalf(
			"expected body %q, got %q",
			expectedBody,
			recorder.Body.String(),
		)
	}

	if recorder.Body.String() == `{"error":"database password leaked"}`+"\n" {
		t.Fatal("internal error details must not be exposed")
	}
}

// TestWriteError_SetsJSONContentType verifies that every translated error
// response is returned as JSON.
func TestWriteError_SetsJSONContentType(t *testing.T) {
	recorder := httptest.NewRecorder()

	presentation_errors.WriteError(
		recorder,
		application_errors.ErrUserNotFound,
	)

	contentType := recorder.Header().Get("Content-Type")

	if contentType != "application/json" {
		t.Fatalf(
			"expected Content-Type application/json, got %q",
			contentType,
		)
	}
}
