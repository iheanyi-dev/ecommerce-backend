package presentation_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
	"github.com/stretchr/testify/assert"
)

const mockRouterAuthenticatedUserID = "550e8400-e29b-41d4-a716-446655440000"

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

// mockRouterUpdateUserProfileService is a test double for the profile-update
// application service.
//
// The router tests only need a concrete UpdateUserProfileHandler so that
// the PATCH /api/v1/users/me route can be registered.
type mockRouterUpdateUserProfileService struct {
	called   bool
	userID   string
	command  dto.UpdateUserProfileCommand
}

func (m *mockRouterUpdateUserProfileService) Execute(
	ctx context.Context,
	userID string,
	command dto.UpdateUserProfileCommand,
) (dto.UpdateUserProfileResult, error) {
	m.called = true
	m.userID = userID
	m.command = command

	return dto.UpdateUserProfileResult{
		UserID:    userID,
		FullName:  command.FullName,
		Email:     "john@example.com",
		Role:      "user",
		Status:    "active",
		UpdatedAt: "2026-09-06T12:00:00.000Z",
	}, nil
}

var _ ports.UpdateUserProfileService = (*mockRouterUpdateUserProfileService)(nil)

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

// mockRouterRefreshUserService is a test double for the refresh-token
// application service.
//
// The router test only needs a concrete RefreshUserHandler so that the
// refresh route can be registered. The actual refresh behavior is tested
// separately by the refresh handler and application tests.
type mockRouterRefreshUserService struct{}

func (m *mockRouterRefreshUserService) Refresh(
	ctx context.Context,
	command dto.RefreshTokenCommand,
) (dto.RefreshTokenResult, error) {
	return dto.RefreshTokenResult{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
	}, nil
}

// mockTokenService is a test double for the application's TokenService.
//
// The router tests do not need the real JWT implementation. They only need
// a TokenService implementation so the authentication middleware can be
// constructed without depending on infrastructure.
type mockTokenService struct {
	role string
	err  error
}

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
	if m.err != nil {
		return ports.AuthenticatedIdentity{}, m.err
	}

	return ports.AuthenticatedIdentity{
		UserID: "550e8400-e29b-41d4-a716-446655440000",
		Role:   m.role,
	}, nil
}

// mockRouterLogoutUserService is a test double for the logout
// application service.
//
// The router tests only need a LogoutUserHandler that can be
// constructed. The actual logout behavior is tested separately.
type mockRouterLogoutUserService struct {
	logoutToken  string
	logoutCalled bool
	err          error
}

func (m *mockRouterLogoutUserService) Logout(
	ctx context.Context,
	refreshToken string,
) error {
	m.logoutCalled = true
	m.logoutToken = refreshToken

	return m.err
}

var _ ports.LogoutUserService = (*mockRouterLogoutUserService)(nil)

func TestNewRouter_RegisterUser(t *testing.T) {
	service := &mockRouterRegisterUserService{}

	registerUserHandler := handlers.NewRegisterUserHandler(service)

	loginUserHandler := handlers.NewLoginUserHandler(
		&mockRouterAuthenticateUserService{},
	)

	refreshUserHandler := handlers.NewRefreshUserHandler(
		&mockRouterRefreshUserService{},
	)

	meHandler := handlers.NewMeHandler()

	updateUserProfileHandler := handlers.NewUpdateUserProfileHandler(
		&mockRouterUpdateUserProfileService{},
	)

	authenticationMiddleware := middleware.NewAuthenticationMiddleware(
		&mockTokenService{
			role: "user",
		},
	)

	logoutUserHandler := handlers.NewLogoutUserHandler(
		&mockRouterLogoutUserService{},
	)

	router := presentation.NewRouter(
		registerUserHandler,
		loginUserHandler,
		refreshUserHandler,
		meHandler,
		updateUserProfileHandler,
		authenticationMiddleware,
		logoutUserHandler,
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

	refreshUserHandler := handlers.NewRefreshUserHandler(
		&mockRouterRefreshUserService{},
	)

	meHandler := handlers.NewMeHandler()

	authenticationMiddleware := middleware.NewAuthenticationMiddleware(
		&mockTokenService{
			role: "user",
		},
	)

	logoutUserHandler := handlers.NewLogoutUserHandler(
		&mockRouterLogoutUserService{},
	)

	updateUserProfileHandler := handlers.NewUpdateUserProfileHandler(
		&mockRouterUpdateUserProfileService{},
	)

	router := presentation.NewRouter(
		registerUserHandler,
		loginUserHandler,
		refreshUserHandler,
		meHandler,
		updateUserProfileHandler,
		authenticationMiddleware,
		logoutUserHandler,
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

	refreshUserHandler := handlers.NewRefreshUserHandler(
		&mockRouterRefreshUserService{},
	)

	meHandler := handlers.NewMeHandler()

	authenticationMiddleware := middleware.NewAuthenticationMiddleware(
		&mockTokenService{
			role: "user",
		},
	)

	logoutUserHandler := handlers.NewLogoutUserHandler(
		&mockRouterLogoutUserService{},
	)

	updateUserProfileHandler := handlers.NewUpdateUserProfileHandler(
		&mockRouterUpdateUserProfileService{},
	)

	router := presentation.NewRouter(
		registerUserHandler,
		loginUserHandler,
		refreshUserHandler,
		meHandler,
		updateUserProfileHandler,
		authenticationMiddleware,
		logoutUserHandler,
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

// TestNewRouter_MeRequiresAuthentication verifies that the /me endpoint
// rejects requests that do not contain an access token.
func TestNewRouter_MeRequiresAuthentication(t *testing.T) {
	t.Parallel()

	registerUserHandler := handlers.NewRegisterUserHandler(
		&mockRouterRegisterUserService{},
	)

	loginUserHandler := handlers.NewLoginUserHandler(
		&mockRouterAuthenticateUserService{},
	)

	refreshUserHandler := handlers.NewRefreshUserHandler(
		&mockRouterRefreshUserService{},
	)

	meHandler := handlers.NewMeHandler()

	authenticationMiddleware := middleware.NewAuthenticationMiddleware(
		&mockTokenService{
			role: "user",
		},
	)

	logoutUserHandler := handlers.NewLogoutUserHandler(
		&mockRouterLogoutUserService{},
	)

	updateUserProfileHandler := handlers.NewUpdateUserProfileHandler(
		&mockRouterUpdateUserProfileService{},
	)

	router := presentation.NewRouter(
		registerUserHandler,
		loginUserHandler,
		refreshUserHandler,
		meHandler,
		updateUserProfileHandler,
		authenticationMiddleware,
		logoutUserHandler,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/me",
		nil,
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	assert.Equal(
		t,
		http.StatusUnauthorized,
		recorder.Code,
	)
}

// TestNewRouter_MeAllowsAuthenticatedUser verifies that an authenticated
// user can access their own profile.
func TestNewRouter_MeAllowsAuthenticatedUser(t *testing.T) {
	t.Parallel()

	registerUserHandler := handlers.NewRegisterUserHandler(
		&mockRouterRegisterUserService{},
	)

	loginUserHandler := handlers.NewLoginUserHandler(
		&mockRouterAuthenticateUserService{},
	)

	refreshUserHandler := handlers.NewRefreshUserHandler(
		&mockRouterRefreshUserService{},
	)

	meHandler := handlers.NewMeHandler()

	authenticationMiddleware := middleware.NewAuthenticationMiddleware(
		&mockTokenService{
			role: "user",
		},
	)

	logoutUserHandler := handlers.NewLogoutUserHandler(
		&mockRouterLogoutUserService{},
	)

	updateUserProfileHandler := handlers.NewUpdateUserProfileHandler(
		&mockRouterUpdateUserProfileService{},
	)

	router := presentation.NewRouter(
		registerUserHandler,
		loginUserHandler,
		refreshUserHandler,
		meHandler,
		updateUserProfileHandler,
		authenticationMiddleware,
		logoutUserHandler,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/me",
		nil,
	)

	req.Header.Set(
		"Authorization",
		"Bearer test-access-token",
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	assert.Equal(
		t,
		http.StatusOK,
		recorder.Code,
	)
}

// TestNewRouter_MeAllowsAuthenticatedVendor verifies that a vendor can
// access the authenticated user's endpoint.
func TestNewRouter_MeAllowsAuthenticatedVendor(t *testing.T) {
	t.Parallel()

	registerUserHandler := handlers.NewRegisterUserHandler(
		&mockRouterRegisterUserService{},
	)

	loginUserHandler := handlers.NewLoginUserHandler(
		&mockRouterAuthenticateUserService{},
	)

	refreshUserHandler := handlers.NewRefreshUserHandler(
		&mockRouterRefreshUserService{},
	)

	meHandler := handlers.NewMeHandler()

	authenticationMiddleware := middleware.NewAuthenticationMiddleware(
		&mockTokenService{
			role: "vendor",
		},
	)

	logoutUserHandler := handlers.NewLogoutUserHandler(
		&mockRouterLogoutUserService{},
	)

	updateUserProfileHandler := handlers.NewUpdateUserProfileHandler(
		&mockRouterUpdateUserProfileService{},
	)

	router := presentation.NewRouter(
		registerUserHandler,
		loginUserHandler,
		refreshUserHandler,
		meHandler,
		updateUserProfileHandler,
		authenticationMiddleware,
		logoutUserHandler,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/me",
		nil,
	)

	req.Header.Set(
		"Authorization",
		"Bearer test-access-token",
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	assert.Equal(
		t,
		http.StatusOK,
		recorder.Code,
	)
}

// TestNewRouter_MeAllowsAuthenticatedAdmin verifies that an admin can
// access the authenticated user's endpoint.
func TestNewRouter_MeAllowsAuthenticatedAdmin(t *testing.T) {
	t.Parallel()

	registerUserHandler := handlers.NewRegisterUserHandler(
		&mockRouterRegisterUserService{},
	)

	loginUserHandler := handlers.NewLoginUserHandler(
		&mockRouterAuthenticateUserService{},
	)

	refreshUserHandler := handlers.NewRefreshUserHandler(
		&mockRouterRefreshUserService{},
	)

	meHandler := handlers.NewMeHandler()

	authenticationMiddleware := middleware.NewAuthenticationMiddleware(
		&mockTokenService{
			role: "admin",
		},
	)

	logoutUserHandler := handlers.NewLogoutUserHandler(
		&mockRouterLogoutUserService{},
	)

	updateUserProfileHandler := handlers.NewUpdateUserProfileHandler(
		&mockRouterUpdateUserProfileService{},
	)

	router := presentation.NewRouter(
		registerUserHandler,
		loginUserHandler,
		refreshUserHandler,
		meHandler,
		updateUserProfileHandler,
		authenticationMiddleware,
		logoutUserHandler,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/me",
		nil,
	)

	req.Header.Set(
		"Authorization",
		"Bearer test-access-token",
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	assert.Equal(
		t,
		http.StatusOK,
		recorder.Code,
	)
}

// TestNewRouter_MeRejectsUnknownRole verifies that authentication alone is
// not sufficient when the caller has a role that is not authorized by the
// route.
func TestNewRouter_MeRejectsUnknownRole(t *testing.T) {
	t.Parallel()

	registerUserHandler := handlers.NewRegisterUserHandler(
		&mockRouterRegisterUserService{},
	)

	loginUserHandler := handlers.NewLoginUserHandler(
		&mockRouterAuthenticateUserService{},
	)

	refreshUserHandler := handlers.NewRefreshUserHandler(
		&mockRouterRefreshUserService{},
	)

	meHandler := handlers.NewMeHandler()

	authenticationMiddleware := middleware.NewAuthenticationMiddleware(
		&mockTokenService{
			role: "unknown",
		},
	)

	logoutUserHandler := handlers.NewLogoutUserHandler(
		&mockRouterLogoutUserService{},
	)

	updateUserProfileHandler := handlers.NewUpdateUserProfileHandler(
		&mockRouterUpdateUserProfileService{},
	)

	router := presentation.NewRouter(
		registerUserHandler,
		loginUserHandler,
		refreshUserHandler,
		meHandler,
		updateUserProfileHandler,
		authenticationMiddleware,
		logoutUserHandler,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/me",
		nil,
	)

	req.Header.Set(
		"Authorization",
		"Bearer test-access-token",
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	assert.Equal(
		t,
		http.StatusForbidden,
		recorder.Code,
	)
}

// TestNewRouter_RefreshUser verifies that the refresh endpoint is registered
// and that the request reaches the refresh handler rather than returning 404.
func TestNewRouter_RefreshUser(t *testing.T) {
	t.Parallel()

	registerUserHandler := handlers.NewRegisterUserHandler(
		&mockRouterRegisterUserService{},
	)

	loginUserHandler := handlers.NewLoginUserHandler(
		&mockRouterAuthenticateUserService{},
	)

	refreshUserHandler := handlers.NewRefreshUserHandler(
		&mockRouterRefreshUserService{},
	)

	meHandler := handlers.NewMeHandler()

	authenticationMiddleware := middleware.NewAuthenticationMiddleware(
		&mockTokenService{
			role: "user",
		},
	)

	logoutUserHandler := handlers.NewLogoutUserHandler(
		&mockRouterLogoutUserService{},
	)

	updateUserProfileHandler := handlers.NewUpdateUserProfileHandler(
		&mockRouterUpdateUserProfileService{},
	)

	router := presentation.NewRouter(
		registerUserHandler,
		loginUserHandler,
		refreshUserHandler,
		meHandler,
		updateUserProfileHandler,
		authenticationMiddleware,
		logoutUserHandler,
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/refresh",
		nil,
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	// The empty body is invalid JSON, so the refresh handler should return
	// 400. A 404 would indicate that the route was not registered.
	assert.Equal(
		t,
		http.StatusBadRequest,
		recorder.Code,
	)
}

func TestNewRouter_LogoutUserRequiresAuthentication(t *testing.T) {
	t.Parallel()

	logoutService := &mockRouterLogoutUserService{}
	logoutHandler := handlers.NewLogoutUserHandler(logoutService)

	tokenService := &mockTokenService{
		role: "user",
	}

	authenticationMiddleware := middleware.NewAuthenticationMiddleware(
		tokenService,
	)

	updateUserProfileHandler := handlers.NewUpdateUserProfileHandler(
		&mockRouterUpdateUserProfileService{},
	)
	router := presentation.NewRouter(
		&handlers.RegisterUserHandler{},
		&handlers.LoginUserHandler{},
		&handlers.RefreshUserHandler{},
		&handlers.MeHandler{},
		updateUserProfileHandler,
		authenticationMiddleware,
		logoutHandler,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/logout",
		nil,
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			recorder.Code,
		)
	}

	if logoutService.logoutCalled {
		t.Fatal("expected logout service not to be called")
	}
}

func TestNewRouter_LogoutUserAllowsAuthenticatedRequest(t *testing.T) {
	t.Parallel()

	logoutService := &mockRouterLogoutUserService{}
	logoutHandler := handlers.NewLogoutUserHandler(logoutService)

	tokenService := &mockTokenService{
		role: "user",
	}

	authenticationMiddleware := middleware.NewAuthenticationMiddleware(
		tokenService,
	)

	updateUserProfileHandler := handlers.NewUpdateUserProfileHandler(
		&mockRouterUpdateUserProfileService{},
	)
	router := presentation.NewRouter(
		&handlers.RegisterUserHandler{},
		&handlers.LoginUserHandler{},
		&handlers.RefreshUserHandler{},
		&handlers.MeHandler{},
		updateUserProfileHandler,
		authenticationMiddleware,
		logoutHandler,
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/logout",
		strings.NewReader(
			`{"refresh_token":"valid-refresh-token"}`,
		),
	)

	request.Header.Set(
		"Authorization",
		"Bearer valid-access-token",
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNoContent,
			recorder.Code,
		)
	}

	if !logoutService.logoutCalled {
		t.Fatal("expected logout service to be called")
	}

	if logoutService.logoutToken != "valid-refresh-token" {
		t.Fatalf(
			"expected refresh token %q, got %q",
			"valid-refresh-token",
			logoutService.logoutToken,
		)
	}
}

func TestRouter_PatchMe_UsesAuthenticatedIdentity(t *testing.T) {
	t.Parallel()
	updateService := &mockRouterUpdateUserProfileService{}

	registerUserHandler := handlers.NewRegisterUserHandler(
		&mockRouterRegisterUserService{},
	)

	loginUserHandler := handlers.NewLoginUserHandler(
		&mockRouterAuthenticateUserService{},
	)

	refreshUserHandler := handlers.NewRefreshUserHandler(
		&mockRouterRefreshUserService{},
	)

	meHandler := handlers.NewMeHandler()

	authenticationMiddleware := middleware.NewAuthenticationMiddleware(
		&mockTokenService{
			role: "user",
		},
	)

	logoutUserHandler := handlers.NewLogoutUserHandler(
		&mockRouterLogoutUserService{},
	)

	updateUserProfileHandler := handlers.NewUpdateUserProfileHandler(
		updateService,
	)

	router := presentation.NewRouter(
		registerUserHandler,
		loginUserHandler,
		refreshUserHandler,
		meHandler,
		updateUserProfileHandler,
		authenticationMiddleware,
		logoutUserHandler,
	)

	body := strings.NewReader(`{
		"user_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		"full_name": "Updated User"
	}`)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/users/me",
		body,
	)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-token")

	rec := httptest.NewRecorder()

	// Act.
	router.ServeHTTP(rec, req)

	// Assert.
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if !updateService.called {
		t.Fatal("expected update user profile service to be called")
	}

	if updateService.userID != mockRouterAuthenticatedUserID {
		t.Fatalf(
			"expected authenticated user ID %q, got %q",
			mockRouterAuthenticatedUserID,
			updateService.userID,
		)
	}

	if updateService.command.FullName != "Updated User" {
		t.Fatalf(
			"expected full name %q, got %q",
			"Updated User",
			updateService.command.FullName,
		)
	}
}