package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/schemas"
)

type mockRefreshUserService struct {
	refreshFunc func(
		ctx context.Context,
		command dto.RefreshTokenCommand,
	) (dto.RefreshTokenResult, error)

	called bool
}

// Refresh implements the application-layer RefreshUserService port.
//
// The mock allows the HTTP handler to be tested independently from the
// refresh-token use case, PostgreSQL, hashing, and JWT infrastructure.
func (m *mockRefreshUserService) Refresh(
	ctx context.Context,
	command dto.RefreshTokenCommand,
) (dto.RefreshTokenResult, error) {
	m.called = true

	return m.refreshFunc(ctx, command)
}

// -----------------------------------------------------------------------------
// Success
// -----------------------------------------------------------------------------

func TestRefreshUserHandler_Success(t *testing.T) {
	service := &mockRefreshUserService{
		refreshFunc: func(
			ctx context.Context,
			command dto.RefreshTokenCommand,
		) (dto.RefreshTokenResult, error) {

			if command.RefreshToken != "old-refresh-token" {
				t.Errorf(
					"expected refresh token %q, got %q",
					"old-refresh-token",
					command.RefreshToken,
				)
			}

			return dto.RefreshTokenResult{
				AccessToken:  "new-access-token",
				RefreshToken: "new-refresh-token",
			}, nil
		},
	}

	handler := handlers.NewRefreshUserHandler(service)

	body := schemas.RefreshTokenRequest{
		RefreshToken: "old-refresh-token",
	}

	requestBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/refresh",
		bytes.NewReader(requestBody),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusOK,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf(
			"expected Content-Type %q, got %q",
			"application/json",
			contentType,
		)
	}

	var response schemas.RefreshTokenResponse

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf(
			"failed to decode response: %v",
			err,
		)
	}

	if response.AccessToken != "new-access-token" {
		t.Errorf(
			"expected access token %q, got %q",
			"new-access-token",
			response.AccessToken,
		)
	}

	if response.RefreshToken != "new-refresh-token" {
		t.Errorf(
			"expected refresh token %q, got %q",
			"new-refresh-token",
			response.RefreshToken,
		)
	}
}

// -----------------------------------------------------------------------------
// Invalid JSON
// -----------------------------------------------------------------------------

func TestRefreshUserHandler_InvalidJSON(t *testing.T) {
	service := &mockRefreshUserService{
		refreshFunc: func(
			ctx context.Context,
			command dto.RefreshTokenCommand,
		) (dto.RefreshTokenResult, error) {
			t.Fatal(
				"application service should not be called for invalid JSON",
			)

			return dto.RefreshTokenResult{}, nil
		},
	}

	handler := handlers.NewRefreshUserHandler(service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/refresh",
		strings.NewReader(`{"refresh_token":`),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}

	if service.called {
		t.Fatal(
			"application service should not be called for invalid JSON",
		)
	}
}

// -----------------------------------------------------------------------------
// HTTP method
// -----------------------------------------------------------------------------

func TestRefreshUserHandler_MethodNotAllowed(t *testing.T) {
	service := &mockRefreshUserService{
		refreshFunc: func(
			ctx context.Context,
			command dto.RefreshTokenCommand,
		) (dto.RefreshTokenResult, error) {
			t.Fatal(
				"application service should not be called for invalid HTTP method",
			)

			return dto.RefreshTokenResult{}, nil
		},
	}

	handler := handlers.NewRefreshUserHandler(service)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/refresh",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusMethodNotAllowed,
			recorder.Code,
		)
	}
}

// -----------------------------------------------------------------------------
// Invalid refresh token
// -----------------------------------------------------------------------------

func TestRefreshUserHandler_InvalidRefreshToken(t *testing.T) {
	service := &mockRefreshUserService{
		refreshFunc: func(
			ctx context.Context,
			command dto.RefreshTokenCommand,
		) (dto.RefreshTokenResult, error) {
			return dto.RefreshTokenResult{}, use_cases.ErrInvalidRefreshToken
		},
	}

	handler := handlers.NewRefreshUserHandler(service)

	body := `{
		"refresh_token": "invalid-refresh-token"
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/refresh",
		strings.NewReader(body),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusUnauthorized,
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

// -----------------------------------------------------------------------------
// Rotation/persistence failure
// -----------------------------------------------------------------------------

func TestRefreshUserHandler_RotationFailure(t *testing.T) {
	service := &mockRefreshUserService{
		refreshFunc: func(
			ctx context.Context,
			command dto.RefreshTokenCommand,
		) (dto.RefreshTokenResult, error) {
			return dto.RefreshTokenResult{}, use_cases.ErrRefreshTokenPersistence
		},
	}

	handler := handlers.NewRefreshUserHandler(service)

	body := `{
		"refresh_token": "valid-refresh-token"
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/refresh",
		strings.NewReader(body),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusInternalServerError,
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

// -----------------------------------------------------------------------------
// Unexpected application error
// -----------------------------------------------------------------------------

func TestRefreshUserHandler_InternalError(t *testing.T) {
	service := &mockRefreshUserService{
		refreshFunc: func(
			ctx context.Context,
			command dto.RefreshTokenCommand,
		) (dto.RefreshTokenResult, error) {
			return dto.RefreshTokenResult{}, errors.New(
				"database connection failed",
			)
		},
	}

	handler := handlers.NewRefreshUserHandler(service)

	body := `{
		"refresh_token": "valid-refresh-token"
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/refresh",
		strings.NewReader(body),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusInternalServerError,
			recorder.Code,
			recorder.Body.String(),
		)
	}
}