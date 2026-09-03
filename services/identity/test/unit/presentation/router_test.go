package presentation_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
	"github.com/stretchr/testify/assert"
)

// mockRouterRegisterUserService is a test double for the registration
// application service.
//
// The router test does not need the real registration use case because
// the purpose of this test is only to verify HTTP route registration.
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

// mockRouterAuthenticateUserService is a test double for the authentication
// application service.
//
// The router only needs an implementation of the application port so that
// the login handler can be constructed.
type mockRouterAuthenticateUserService struct{}

func (m *mockRouterAuthenticateUserService) Authenticate(
	ctx context.Context,
	command dto.LoginUserCommand,
) (dto.LoginUserResult, error) {
	return dto.LoginUserResult{
		ID:          "550e8400-e29b-41d4-a716-446655440000",
		Email:       "john@example.com",
		Role:        "user",
		Status:      "active",
		AccessToken: "test-access-token",
	}, nil
}

// mockTokenService is a test double for the application's TokenService.
//
// The router tests do not need the real JWT implementation. They only need
// a TokenService implementation so the authentication middleware can be
// constructed without depending on infrastructure.
type mockTokenService struct{}

func (m *mockTokenService) GenerateAccessToken(
	ctx context.Context,
	userID user.UserID,
	role user.Role,
) (string, error) {
	return "test-access-token", nil
}

func (m *mockTokenService) ValidateAccessToken(
	ctx context.Context,
	token string,
) (ports.AuthenticatedIdentity, error) {
	return ports.AuthenticatedIdentity{
		UserID: "550e8400-e29b-41d4-a716-446655440000",
		Role:   "user",
	}, nil
}

func TestNewRouter_RegisterUser(t *testing.T) {
	service := &mockRouterRegisterUserService{}

	registerUserHandler := handlers.NewRegisterUserHandler(service)

	loginUserHandler := handlers.NewLoginUserHandler(
		&mockRouterAuthenticateUserService{},
	)

	meHandler := handlers.NewMeHandler()

	authenticationMiddleware := middleware.NewAuthenticationMiddleware(
		&mockTokenService{},
	)

	router := presentation.NewRouter(
		registerUserHandler,
		loginUserHandler,
		meHandler,
		authenticationMiddleware,
	)

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

	loginUserHandler := handlers.NewLoginUserHandler(
		&mockRouterAuthenticateUserService{},
	)

	meHandler := handlers.NewMeHandler()

	authenticationMiddleware := middleware.NewAuthenticationMiddleware(
		&mockTokenService{},
	)

	router := presentation.NewRouter(
		registerUserHandler,
		loginUserHandler,
		meHandler,
		authenticationMiddleware,
	)

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

// TestNewRouter_LoginUser verifies that the login endpoint is registered
// and that the request reaches the login handler rather than returning 404.
func TestNewRouter_LoginUser(t *testing.T) {
	t.Parallel()

	registerUserHandler := handlers.NewRegisterUserHandler(
		&mockRouterRegisterUserService{},
	)

	loginUserHandler := handlers.NewLoginUserHandler(
		&mockRouterAuthenticateUserService{},
	)

	meHandler := handlers.NewMeHandler()

	authenticationMiddleware := middleware.NewAuthenticationMiddleware(
		&mockTokenService{},
	)

	router := presentation.NewRouter(
		registerUserHandler,
		loginUserHandler,
		meHandler,
		authenticationMiddleware,
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/login",
		nil,
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	// The empty body is invalid JSON, so the login handler should return
	// 400. A 404 would indicate that the route was not registered.
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}
