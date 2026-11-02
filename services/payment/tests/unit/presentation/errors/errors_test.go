// Package errors_test verifies the HTTP error translation contract exposed
// by the Payment presentation layer.
package errors_test

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	application "github.com/iheanyi-dev/ecommerce-backend/services/payment/application"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/errors"
	presentationerrors "github.com/iheanyi-dev/ecommerce-backend/services/payment/presentation/errors"
)

// errorResponse represents the public JSON response contract expected from
// the Payment presentation error translator.
type errorResponse struct {
	Error string `json:"error"`
}

// TestWriteError verifies that application, domain, and presentation errors
// are translated into the correct HTTP status and stable public message.
// Each case wraps its error to ensure classification uses errors.Is rather
// than direct error equality.
func TestWriteError(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		expectedStatus  int
		expectedMessage string
	}{
		{
			name:            "payment not found",
			err:             application.ErrPaymentNotFound,
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "payment not found",
		},
		{
			name:            "payment attempt not found",
			err:             application.ErrPaymentAttemptNotFound,
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "payment attempt not found",
		},
		{
			name:            "payout not found",
			err:             application.ErrPayoutNotFound,
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "payout not found",
		},
		{
			name:            "payout attempt not found",
			err:             application.ErrPayoutAttemptNotFound,
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "payout attempt not found",
		},
		{
			name:            "payment already exists",
			err:             application.ErrPaymentAlreadyExists,
			expectedStatus:  http.StatusConflict,
			expectedMessage: "payment already exists",
		},
		{
			name:            "payment idempotency conflict",
			err:             application.ErrIdempotencyConflict,
			expectedStatus:  http.StatusConflict,
			expectedMessage: "payment reference conflict",
		},
		{
			name:            "payout already exists",
			err:             application.ErrPayoutAlreadyExists,
			expectedStatus:  http.StatusConflict,
			expectedMessage: "payout already exists",
		},
		{
			name:            "payout idempotency conflict",
			err:             application.ErrPayoutIdempotencyConflict,
			expectedStatus:  http.StatusConflict,
			expectedMessage: "payout reference conflict",
		},
		{
			name:            "attempt requires reconciliation",
			err:             application.ErrAttemptRequiresReconciliation,
			expectedStatus:  http.StatusConflict,
			expectedMessage: "payment attempt requires reconciliation",
		},
		{
			name:            "payout retry not allowed",
			err:             application.ErrPayoutRetryNotAllowed,
			expectedStatus:  http.StatusConflict,
			expectedMessage: "payout retry not allowed",
		},
		{
			name:            "invalid pagination",
			err:             application.ErrInvalidPagination,
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "invalid pagination",
		},
		{
			name:            "invalid provider response",
			err:             application.ErrInvalidProviderResponse,
			expectedStatus:  http.StatusBadGateway,
			expectedMessage: "invalid payment provider response",
		},
		{
			name:            "invalid payment",
			err:             domainerrors.ErrInvalidPayment,
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "invalid request",
		},
		{
			name:            "invalid payment transition",
			err:             domainerrors.ErrInvalidPaymentTransition,
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "invalid request",
		},
		{
			name:            "invalid money",
			err:             domainerrors.ErrInvalidMoney,
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "invalid request",
		},
		{
			name:            "invalid business reference",
			err:             domainerrors.ErrInvalidBusinessReference,
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "invalid request",
		},
		{
			name:            "invalid payment attempt",
			err:             domainerrors.ErrInvalidPaymentAttempt,
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "invalid request",
		},
		{
			name:            "invalid payment attempt transition",
			err:             domainerrors.ErrInvalidPaymentAttemptTransition,
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "invalid request",
		},
		{
			name:            "invalid payout",
			err:             domainerrors.ErrInvalidPayout,
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "invalid request",
		},
		{
			name:            "invalid payout transition",
			err:             domainerrors.ErrInvalidPayoutTransition,
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "invalid request",
		},
		{
			name:            "invalid payout attempt",
			err:             domainerrors.ErrInvalidPayoutAttempt,
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "invalid request",
		},
		{
			name:            "invalid payout attempt transition",
			err:             domainerrors.ErrInvalidPayoutAttemptTransition,
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "invalid request",
		},
		{
			name:            "authentication required",
			err:             presentationerrors.ErrAuthenticationRequired,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "authentication required",
		},
		{
			name:            "forbidden",
			err:             presentationerrors.ErrForbidden,
			expectedStatus:  http.StatusForbidden,
			expectedMessage: "forbidden",
		},
		{
			name:            "method not allowed",
			err:             presentationerrors.ErrMethodNotAllowed,
			expectedStatus:  http.StatusMethodNotAllowed,
			expectedMessage: "method not allowed",
		},
		{
			name:            "invalid request body",
			err:             presentationerrors.ErrInvalidRequestBody,
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "invalid request body",
		},
		{
			name:            "invalid multipart request",
			err:             presentationerrors.ErrInvalidMultipartRequest,
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "invalid request body",
		},
		{
			name:            "invalid service credentials",
			err:             presentationerrors.ErrInvalidServiceCredentials,
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "invalid service credentials",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", nil)
			recorder := httptest.NewRecorder()

			wrappedErr := fmt.Errorf("operation failed: %w", test.err)
			presentationerrors.WriteError(recorder, wrappedErr)

			if recorder.Code != test.expectedStatus {
				t.Fatalf(
					"expected HTTP status %d, got %d",
					test.expectedStatus,
					recorder.Code,
				)
			}

			if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
				t.Fatalf(
					"expected Content-Type application/json, got %q",
					contentType,
				)
			}

			var response errorResponse
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("decode error response: %v", err)
			}

			if response.Error != test.expectedMessage {
				t.Errorf(
					"expected error message %q, got %q",
					test.expectedMessage,
					response.Error,
				)
			}

			_ = request
		})
	}
}

// TestWriteErrorDefaultsToInternalServerError verifies that unrecognized
// errors never expose their internal details in the public HTTP response.
func TestWriteErrorDefaultsToInternalServerError(t *testing.T) {
	recorder := httptest.NewRecorder()
	internalErr := stderrors.New("sensitive database connection details")

	presentationerrors.WriteError(recorder, internalErr)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected HTTP status %d, got %d",
			http.StatusInternalServerError,
			recorder.Code,
		)
	}

	var response errorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if response.Error != "internal server error" {
		t.Errorf(
			"expected generic internal error message, got %q",
			response.Error,
		)
	}
}
