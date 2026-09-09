package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
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

// Authenticate implements the authentication service port for tests.
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
	assert.Equal(
		t,
		"application/json",
		recorder.Header().Get("Content-Type"),
	)

	assert.Equal(
		t,
		"{\"error\":\"invalid request body\"}\n",
		recorder.Body.String(),
	)
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
	assert.Equal(
		t,
		"application/json",
		recorder.Header().Get("Content-Type"),
	)

	assert.Equal(
		t,
		"{\"error\":\"method not allowed\"}\n",
		recorder.Body.String(),
	)
}

// TestLoginUserHandler_InvalidCredentials verifies that authentication
// failures are translated into the standardized JSON error response.
func TestLoginUserHandler_InvalidCredentials(t *testing.T) {
	t.Parallel()

	mockService := &mockAuthenticateUserService{
		authenticateFunc: func(
			cmd dto.LoginUserCommand,
		) (dto.LoginUserResult, error) {
			return dto.LoginUserResult{}, application_errors.ErrInvalidCredentials
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
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	assert.JSONEq(
		t,
		`{"error":"invalid credentials"}`,
		recorder.Body.String(),
	)
}

// TestLoginUserHandler_InternalError verifies that unexpected application
// failures are returned as a standardized internal server error without
// exposing implementation details to the client.
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

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

	assert.JSONEq(
		t,
		`{"error":"internal server error"}`,
		recorder.Body.String(),
	)

	assert.NotContains(
		t,
		recorder.Body.String(),
		"database unavailable",
	)
}

// TestLoginUserHandler_AccountNotActive verifies that an inactive account
// cannot obtain an access token and that the standardized error response
// is returned.
func TestLoginUserHandler_AccountNotActive(t *testing.T) {
	t.Parallel()

	mockService := &mockAuthenticateUserService{
		authenticateFunc: func(
			cmd dto.LoginUserCommand,
		) (dto.LoginUserResult, error) {
			return dto.LoginUserResult{}, application_errors.ErrAccountNotActive
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
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	assert.JSONEq(
		t,
		`{"error":"account is not active"}`,
		recorder.Body.String(),
	)
}

// TestLoginUserHandler_TokenGenerationError verifies that token-generation
// failures are hidden behind the common internal-server-error response.
func TestLoginUserHandler_TokenGenerationError(t *testing.T) {
	t.Parallel()

	mockService := &mockAuthenticateUserService{
		authenticateFunc: func(
			cmd dto.LoginUserCommand,
		) (dto.LoginUserResult, error) {
			return dto.LoginUserResult{}, application_errors.ErrTokenGeneration
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

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	assert.JSONEq(
		t,
		`{"error":"internal server error"}`,
		recorder.Body.String(),
	)
}
