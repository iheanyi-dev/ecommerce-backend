package presentation_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/handlers"
)

type mockRouterRegisterUserService struct{}

func (m *mockRouterRegisterUserService) Execute(
	ctx context.Context,
	command dto.RegisterUserCommand,
) (dto.RegisterUserResult, error) {
	return dto.RegisterUserResult{
		ID:       "550e8400-e29b-41d4-a716-446655440000",
		FullName: "John Doe",
		Email:    "john@example.com",
		Role:     "user",
		Status:   "pending_verification",
	}, nil
}

func TestNewRouter_RegisterUser(t *testing.T) {
	service := &mockRouterRegisterUserService{}

	registerUserHandler := handlers.NewRegisterUserHandler(service)

	router := presentation.NewRouter(registerUserHandler)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/register",
		nil,
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	// A nil body is invalid JSON, so the request should reach the
	// registration handler and return 400 rather than 404.
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestNewRouter_UnknownRoute(t *testing.T) {
	service := &mockRouterRegisterUserService{}

	registerUserHandler := handlers.NewRegisterUserHandler(service)

	router := presentation.NewRouter(registerUserHandler)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/unknown",
		nil,
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			recorder.Code,
		)
	}
}