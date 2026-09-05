package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/handlers"
)

type mockRegisterUserService struct {
	executeFunc func(
		ctx context.Context,
		command dto.RegisterUserCommand,
	) (dto.RegisterUserResult, error)
}

// Execute implements the RegisterUserService interface used by the
// registration handler.
//
// The mock allows the presentation layer to be tested without involving
// PostgreSQL, SQLC, password hashing, or any other infrastructure concern.
func (m *mockRegisterUserService) Execute(
	ctx context.Context,
	command dto.RegisterUserCommand,
) (dto.RegisterUserResult, error) {
	return m.executeFunc(ctx, command)
}

// -----------------------------------------------------------------------------
// Success
// -----------------------------------------------------------------------------

func TestRegisterUserHandler_Success(t *testing.T) {
	createdAt := time.Now().UTC()
	updatedAt := createdAt

	service := &mockRegisterUserService{
		executeFunc: func(
			ctx context.Context,
			command dto.RegisterUserCommand,
		) (dto.RegisterUserResult, error) {

			if command.FullName != "John Doe" {
				t.Errorf(
					"expected full name %q, got %q",
					"John Doe",
					command.FullName,
				)
			}

			if command.Email != "john@example.com" {
				t.Errorf(
					"expected email %q, got %q",
					"john@example.com",
					command.Email,
				)
			}

			if command.Password != "SecurePassword123" {
				t.Errorf(
					"expected password %q, got %q",
					"SecurePassword123",
					command.Password,
				)
			}

			return dto.RegisterUserResult{
				ID:        "550e8400-e29b-41d4-a716-446655440000",
				FullName:  "John Doe",
				Email:     "john@example.com",
				Role:      "user",
				Status:    "pending_verification",
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			}, nil
		},
	}

	handler := handlers.NewRegisterUserHandler(service)

	body := `{
		"full_name": "John Doe",
		"email": "john@example.com",
		"password": "SecurePassword123"
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/register",
		strings.NewReader(body),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusCreated,
			recorder.Code,
		)
	}

	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf(
			"expected Content-Type %q, got %q",
			"application/json",
			contentType,
		)
	}

	var response struct {
		ID        string    `json:"id"`
		FullName  string    `json:"full_name"`
		Email     string    `json:"email"`
		Role      string    `json:"role"`
		Status    string    `json:"status"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf(
			"failed to decode response: %v",
			err,
		)
	}

	if response.ID == "" {
		t.Error("expected response to contain user ID")
	}

	if response.FullName != "John Doe" {
		t.Errorf(
			"expected full name %q, got %q",
			"John Doe",
			response.FullName,
		)
	}

	if response.Email != "john@example.com" {
		t.Errorf(
			"expected email %q, got %q",
			"john@example.com",
			response.Email,
		)
	}

	if response.Role != "user" {
		t.Errorf(
			"expected role %q, got %q",
			"user",
			response.Role,
		)
	}

	if response.Status != "pending_verification" {
		t.Errorf(
			"expected status %q, got %q",
			"pending_verification",
			response.Status,
		)
	}
}

// -----------------------------------------------------------------------------
// Invalid JSON
// -----------------------------------------------------------------------------

func TestRegisterUserHandler_InvalidJSON(t *testing.T) {
	service := &mockRegisterUserService{
		executeFunc: func(
			ctx context.Context,
			command dto.RegisterUserCommand,
		) (dto.RegisterUserResult, error) {
			t.Fatal(
				"application service should not be called for invalid JSON",
			)

			return dto.RegisterUserResult{}, nil
		},
	}

	handler := handlers.NewRegisterUserHandler(service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/register",
		strings.NewReader(`{"full_name": "John Doe",`),
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
}

// -----------------------------------------------------------------------------
// HTTP method
// -----------------------------------------------------------------------------

func TestRegisterUserHandler_MethodNotAllowed(t *testing.T) {
	service := &mockRegisterUserService{
		executeFunc: func(
			ctx context.Context,
			command dto.RegisterUserCommand,
		) (dto.RegisterUserResult, error) {
			t.Fatal(
				"application service should not be called for invalid HTTP method",
			)

			return dto.RegisterUserResult{}, nil
		},
	}

	handler := handlers.NewRegisterUserHandler(service)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/register",
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
// Email already exists
// -----------------------------------------------------------------------------

func TestRegisterUserHandler_EmailAlreadyExists(t *testing.T) {
	service := &mockRegisterUserService{
		executeFunc: func(
			ctx context.Context,
			command dto.RegisterUserCommand,
		) (dto.RegisterUserResult, error) {
			return dto.RegisterUserResult{}, use_cases.ErrEmailAlreadyExists
		},
	}

	handler := handlers.NewRegisterUserHandler(service)

	body := `{
		"full_name": "John Doe",
		"email": "john@example.com",
		"password": "SecurePassword123"
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/register",
		strings.NewReader(body),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusConflict,
			recorder.Code,
		)
	}
}

// -----------------------------------------------------------------------------
// Invalid email
// -----------------------------------------------------------------------------

func TestRegisterUserHandler_InvalidEmail(t *testing.T) {
	service := &mockRegisterUserService{
		executeFunc: func(
			ctx context.Context,
			command dto.RegisterUserCommand,
		) (dto.RegisterUserResult, error) {
			return dto.RegisterUserResult{}, user.ErrInvalidEmail
		},
	}

	handler := handlers.NewRegisterUserHandler(service)

	body := `{
		"full_name": "John Doe",
		"email": "invalid-email",
		"password": "SecurePassword123"
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/register",
		strings.NewReader(body),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusBadRequest,
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

// -----------------------------------------------------------------------------
// Invalid full name
// -----------------------------------------------------------------------------

func TestRegisterUserHandler_InvalidFullName(t *testing.T) {
	service := &mockRegisterUserService{
		executeFunc: func(
			ctx context.Context,
			command dto.RegisterUserCommand,
		) (dto.RegisterUserResult, error) {
			return dto.RegisterUserResult{}, user.ErrInvalidFullName
		},
	}

	handler := handlers.NewRegisterUserHandler(service)

	body := `{
		"full_name": "",
		"email": "john@example.com",
		"password": "SecurePassword123"
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/register",
		strings.NewReader(body),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusBadRequest,
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

// -----------------------------------------------------------------------------
// Internal application/infrastructure error
// -----------------------------------------------------------------------------

func TestRegisterUserHandler_InternalError(t *testing.T) {
	service := &mockRegisterUserService{
		executeFunc: func(
			ctx context.Context,
			command dto.RegisterUserCommand,
		) (dto.RegisterUserResult, error) {
			return dto.RegisterUserResult{}, errors.New(
				"database connection failed",
			)
		},
	}

	handler := handlers.NewRegisterUserHandler(service)

	body := `{
		"full_name": "John Doe",
		"email": "john@example.com",
		"password": "SecurePassword123"
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/register",
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
			"expected status %d, got %d",
			http.StatusInternalServerError,
			recorder.Code,
		)
	}
}