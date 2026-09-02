package presentation_test

import (
	"context"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/schemas"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAuthenticateUserService is a test double for the application-layer
// authentication port.
//
// The presentation layer should depend on the application port rather than
// the concrete AuthenticateUserUseCase implementation.
type mockAuthenticateUserService struct {
	authenticateFunc func(
		cmd dto.LoginUserCommand,
	) (dto.LoginUserResult, error)
}

func (m *mockAuthenticateUserService) Authenticate(
	ctx context.Context,
	cmd dto.LoginUserCommand,
) (dto.LoginUserResult, error) {
	return m.authenticateFunc(cmd)
}

// TestLoginUserHandler_Success verifies that valid credentials are passed
// into the application layer and that the resulting authentication data
// is correctly returned as JSON.
func TestLoginUserHandler_Success(t *testing.T) {
	t.Parallel()

	mockService := &mockAuthenticateUserService{
		authenticateFunc: func(cmd dto.LoginUserCommand) (dto.LoginUserResult, error) {
			assert.Equal(t, "john@example.com", cmd.Email)
			assert.Equal(t, "correct-password", cmd.Password)

			return dto.LoginUserResult{
				ID:          "user-id-123",
				Email:       "john@example.com",
				Role:        "user",
				Status:      "active",
				AccessToken: "jwt-access-token",
			}, nil
		},
	}

	handler := handlers.NewLoginUserHandler(mockService)

	requestBody := schemas.LoginUserRequest{
		Email:    "john@example.com",
		Password: "correct-password",
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/login",
		bytes.NewReader(body),
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var response schemas.LoginUserResponse

	err = json.NewDecoder(recorder.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "user-id-123", response.ID)
	assert.Equal(t, "john@example.com", response.Email)
	assert.Equal(t, "user", response.Role)
	assert.Equal(t, "active", response.Status)
	assert.Equal(t, "jwt-access-token", response.AccessToken)
}

// TestLoginUserHandler_InvalidJSON verifies that malformed JSON is rejected
// at the presentation boundary without invoking the application layer.
func TestLoginUserHandler_InvalidJSON(t *testing.T) {
	t.Parallel()

	serviceCalled := false

	mockService := &mockAuthenticateUserService{
		authenticateFunc: func(
			cmd dto.LoginUserCommand,
		) (dto.LoginUserResult, error) {
			serviceCalled = true

			return dto.LoginUserResult{}, nil
		},
	}

	handler := handlers.NewLoginUserHandler(mockService)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/login",
		bytes.NewBufferString(`{"email":`),
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.False(t, serviceCalled)
}

// TestLoginUserHandler_MethodNotAllowed verifies that the login endpoint
// accepts POST requests only.
func TestLoginUserHandler_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	mockService := &mockAuthenticateUserService{
		authenticateFunc: func(
			cmd dto.LoginUserCommand,
		) (dto.LoginUserResult, error) {
			return dto.LoginUserResult{}, nil
		},
	}

	handler := handlers.NewLoginUserHandler(mockService)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/login",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusMethodNotAllowed, recorder.Code)
}

// TestLoginUserHandler_InvalidCredentials verifies that authentication
// failures are translated into an appropriate HTTP response.
func TestLoginUserHandler_InvalidCredentials(t *testing.T) {
	t.Parallel()

	mockService := &mockAuthenticateUserService{
		authenticateFunc: func(
			cmd dto.LoginUserCommand,
		) (dto.LoginUserResult, error) {
			return dto.LoginUserResult{}, use_cases.ErrInvalidCredentials
		},
	}

	handler := handlers.NewLoginUserHandler(mockService)

	requestBody := schemas.LoginUserRequest{
		Email:    "john@example.com",
		Password: "wrong-password",
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/login",
		bytes.NewReader(body),
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

// TestLoginUserHandler_InternalError verifies that unexpected application
// failures are returned as an internal server error rather than exposing
// implementation details to the client.
func TestLoginUserHandler_InternalError(t *testing.T) {
	t.Parallel()

	mockService := &mockAuthenticateUserService{
		authenticateFunc: func(
			cmd dto.LoginUserCommand,
		) (dto.LoginUserResult, error) {
			return dto.LoginUserResult{}, errors.New("database unavailable")
		},
	}

	handler := handlers.NewLoginUserHandler(mockService)

	requestBody := schemas.LoginUserRequest{
		Email:    "john@example.com",
		Password: "correct-password",
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/login",
		bytes.NewReader(body),
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	// The exact distinction between authentication errors and infrastructure
	// errors will depend on the typed application errors we already defined.
	//
	// For now this test documents the expected presentation behavior.
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

// TestLoginUserHandler_AccountNotActive verifies that an authenticated
// account which is not active cannot obtain an access token.
func TestLoginUserHandler_AccountNotActive(t *testing.T) {
	t.Parallel()

	mockService := &mockAuthenticateUserService{
		authenticateFunc: func(
			cmd dto.LoginUserCommand,
		) (dto.LoginUserResult, error) {
			return dto.LoginUserResult{}, use_cases.ErrAccountNotActive
		},
	}

	handler := handlers.NewLoginUserHandler(mockService)

	requestBody := schemas.LoginUserRequest{
		Email:    "john@example.com",
		Password: "correct-password",
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/login",
		bytes.NewReader(body),
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}