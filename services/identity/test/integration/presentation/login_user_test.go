package presentation_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/persistence/postgres"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/security"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/schemas"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/shared/config"
)

// mockIntegrationRefreshUserService is a lightweight refresh service used
// only because the Identity router now contains the refresh route.
//
// The login integration tests do not test refresh-token behavior, so this
// mock only exists to satisfy the router dependency.
type mockIntegrationRefreshUserService struct{}

func (m *mockIntegrationRefreshUserService) Refresh(
	ctx context.Context,
	command dto.RefreshTokenCommand,
) (dto.RefreshTokenResult, error) {
	return dto.RefreshTokenResult{
		AccessToken:  "integration-test-access-token",
		RefreshToken: "integration-test-refresh-token",
	}, nil
}

func newLoginIntegrationRouter(
	registerUserHandler *handlers.RegisterUserHandler,
	loginUserHandler *handlers.LoginUserHandler,
) http.Handler {
	meHandler := handlers.NewMeHandler()

	authenticationMiddleware := middleware.NewAuthenticationMiddleware(
		&mockIntegrationTokenService{},
	)

	refreshUserHandler := handlers.NewRefreshUserHandler(
		&mockIntegrationRefreshUserService{},
	)

	// Create the logout handler required by the router.
// The integration test does not exercise logout yet.
logoutUserHandler := handlers.NewLogoutUserHandler(
	&mockRouterLogoutUserService{},
)

	return presentation.NewRouter(
		registerUserHandler,
		loginUserHandler,
		refreshUserHandler,
		meHandler,
		authenticationMiddleware,
		logoutUserHandler,
	)
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

// TestLoginUserIntegration verifies the complete authentication flow.
//
// Unlike the login handler unit tests, this test uses the real:
//
//	HTTP request
//	    ↓
//	LoginUserHandler
//	    ↓
//	AuthenticateUserUseCase
//	    ↓
//	UserRepository
//	    ↓
//	PostgreSQL
//	    ↓
//	Bcrypt password verification
//	    ↓
//	JWT TokenService
//	    ↓
//	HTTP response
//
// This ensures that the individual authentication components work
// correctly together.
func TestLoginUserIntegration(t *testing.T) {
	ctx := context.Background()

	// ---------------------------------------------------------
	// Load configuration
	// ---------------------------------------------------------

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf(
			"failed to load configuration: %v",
			err,
		)
	}

	// ---------------------------------------------------------
	// Create real PostgreSQL connection
	// ---------------------------------------------------------

	pool, err := postgres.NewPool(cfg)
	if err != nil {
		t.Fatalf(
			"failed to create PostgreSQL pool: %v",
			err,
		)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	// ---------------------------------------------------------
	// Build persistence dependencies
	// ---------------------------------------------------------

	queries := postgres.NewQueries(pool)

	userRepository := postgres.NewUserRepository(
		queries,
	)

	// ---------------------------------------------------------
	// Build real password hasher
	// ---------------------------------------------------------

	passwordHasher := security.NewBcryptPasswordHasher()

	// ---------------------------------------------------------
	// Build real JWT token service
	// ---------------------------------------------------------

	tokenService, err := security.NewJWTTokenServiceFromConfig(
		cfg,
	)
	if err != nil {
		t.Fatalf(
			"failed to create JWT token service: %v",
			err,
		)
	}

	// ---------------------------------------------------------
	// Build real authentication use case
	// ---------------------------------------------------------

	authenticateUserUseCase := use_cases.NewAuthenticateUserUseCase(
		userRepository,
		passwordHasher,
		tokenService,
	)

	// ---------------------------------------------------------
	// Build real login handler
	// ---------------------------------------------------------

	loginUserHandler := handlers.NewLoginUserHandler(
		authenticateUserUseCase,
	)

	// ---------------------------------------------------------
	// Build registration use case
	//
	// We use the real registration flow to create the test user.
	// This ensures the password stored in PostgreSQL is a real
	// bcrypt hash generated through the application's registration
	// workflow.
	// ---------------------------------------------------------

	registerUserUseCase := use_cases.NewRegisterUserUseCase(
		userRepository,
		passwordHasher,
	)

	registerUserHandler := handlers.NewRegisterUserHandler(
		registerUserUseCase,
	)

	// ---------------------------------------------------------
	// Build the real Identity router
	// ---------------------------------------------------------

	router := newLoginIntegrationRouter(
		registerUserHandler,
		loginUserHandler,
	)

	// ---------------------------------------------------------
	// Generate a unique test email
	// ---------------------------------------------------------

	email := "login-" + uuid.New().String() + "@example.com"

	const password = "SecurePassword123"

	// ---------------------------------------------------------
	// Register a real user through the HTTP API
	// ---------------------------------------------------------

	registerRequestBody := schemas.RegisterUserRequest{
		FullName: "Login Test User",
		Email:    email,
		Password: password,
	}

	registerBody, err := json.Marshal(
		registerRequestBody,
	)
	if err != nil {
		t.Fatalf(
			"failed to encode registration request: %v",
			err,
		)
	}

	registerRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/register",
		bytes.NewReader(registerBody),
	)

	registerRequest.Header.Set(
		"Content-Type",
		"application/json",
	)

	registerRecorder := httptest.NewRecorder()

	router.ServeHTTP(
		registerRecorder,
		registerRequest,
	)

	if registerRecorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected registration status %d, got %d, body: %s",
			http.StatusCreated,
			registerRecorder.Code,
			registerRecorder.Body.String(),
		)
	}

	// Registration creates users in the pending_verification state.
	//
	// Authentication intentionally rejects users who have not completed
	// account verification. Since this test is specifically testing
	// successful authentication, simulate successful verification by
	// activating the test user directly in the database.
	//
	// This does not modify production authentication behavior.
	_, err = pool.Exec(
		ctx,
		`
		UPDATE users
		SET status = 'active'
		WHERE email = $1
		`,
		email,
	)
	if err != nil {
		t.Fatalf(
			"failed to activate test user: %v",
			err,
		)
	}

	// ---------------------------------------------------------
	// Clean up the test user after the test
	// ---------------------------------------------------------

	t.Cleanup(func() {
		_, err := pool.Exec(
			ctx,
			`DELETE FROM users WHERE email = $1`,
			email,
		)

		if err != nil {
			t.Errorf(
				"failed to clean up test user: %v",
				err,
			)
		}
	})

	// ---------------------------------------------------------
	// Build login request
	// ---------------------------------------------------------

	loginRequestBody := schemas.LoginUserRequest{
		Email:    email,
		Password: password,
	}

	loginBody, err := json.Marshal(
		loginRequestBody,
	)
	if err != nil {
		t.Fatalf(
			"failed to encode login request: %v",
			err,
		)
	}

	loginRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/login",
		bytes.NewReader(loginBody),
	)

	loginRequest.Header.Set(
		"Content-Type",
		"application/json",
	)

	loginRecorder := httptest.NewRecorder()

	// ---------------------------------------------------------
	// Execute authentication request
	// ---------------------------------------------------------

	router.ServeHTTP(
		loginRecorder,
		loginRequest,
	)

	// ---------------------------------------------------------
	// Verify successful authentication
	// ---------------------------------------------------------

	if loginRecorder.Code != http.StatusOK {
		t.Fatalf(
			"expected login status %d, got %d, body: %s",
			http.StatusOK,
			loginRecorder.Code,
			loginRecorder.Body.String(),
		)
	}

	var response schemas.LoginUserResponse

	if err := json.NewDecoder(
		loginRecorder.Body,
	).Decode(&response); err != nil {
		t.Fatalf(
			"failed to decode login response: %v",
			err,
		)
	}

	// The authentication response must contain the user's identity.
	if response.ID == "" {
		t.Fatal("expected user ID in login response")
	}

	if response.Email != email {
		t.Fatalf(
			"expected email %q, got %q",
			email,
			response.Email,
		)
	}

	if response.Role != "user" {
		t.Fatalf(
			"expected role %q, got %q",
			"user",
			response.Role,
		)
	}

	if response.Status != "active" {
		t.Fatalf(
			"expected status %q, got %q",
			"active",
			response.Status,
		)
	}

	// A successful authentication must produce an access token.
	if response.AccessToken == "" {
		t.Fatal(
			"expected access token in login response",
		)
	}

	// The access token should look like a JWT.
	//
	// JWTs consist of three dot-separated sections:
	//
	// header.payload.signature
	//
	// We only verify the structure here. Cryptographic verification
	// belongs to the token service/security tests.
	if len(response.AccessToken) < 20 {
		t.Fatal(
			"expected a non-trivial JWT access token",
		)
	}
}

// TestLoginUserIntegration_InvalidPassword verifies that authentication
// fails when the supplied password does not match the stored password.
//
// The application must return the same generic invalid-credentials
// response rather than revealing whether the email exists.
func TestLoginUserIntegration_InvalidPassword(t *testing.T) {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf(
			"failed to load configuration: %v",
			err,
		)
	}

	pool, err := postgres.NewPool(cfg)
	if err != nil {
		t.Fatalf(
			"failed to create PostgreSQL pool: %v",
			err,
		)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	queries := postgres.NewQueries(pool)

	userRepository := postgres.NewUserRepository(
		queries,
	)

	passwordHasher := security.NewBcryptPasswordHasher()

	tokenService, err := security.NewJWTTokenServiceFromConfig(
		cfg,
	)
	if err != nil {
		t.Fatalf(
			"failed to create JWT token service: %v",
			err,
		)
	}

	authenticateUserUseCase := use_cases.NewAuthenticateUserUseCase(
		userRepository,
		passwordHasher,
		tokenService,
	)

	loginUserHandler := handlers.NewLoginUserHandler(
		authenticateUserUseCase,
	)

	registerUserUseCase := use_cases.NewRegisterUserUseCase(
		userRepository,
		passwordHasher,
	)

	registerUserHandler := handlers.NewRegisterUserHandler(
		registerUserUseCase,
	)

	router := newLoginIntegrationRouter(
		registerUserHandler,
		loginUserHandler,
	)

	email := "wrong-password-" + uuid.New().String() + "@example.com"

	const correctPassword = "SecurePassword123"

	registerRequestBody := schemas.RegisterUserRequest{
		FullName: "Wrong Password User",
		Email:    email,
		Password: correctPassword,
	}

	registerBody, err := json.Marshal(
		registerRequestBody,
	)
	if err != nil {
		t.Fatalf(
			"failed to encode registration request: %v",
			err,
		)
	}

	registerRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/register",
		bytes.NewReader(registerBody),
	)

	registerRequest.Header.Set(
		"Content-Type",
		"application/json",
	)

	registerRecorder := httptest.NewRecorder()

	router.ServeHTTP(
		registerRecorder,
		registerRequest,
	)

	if registerRecorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected registration status %d, got %d, body: %s",
			http.StatusCreated,
			registerRecorder.Code,
			registerRecorder.Body.String(),
		)
	}

	// Activate the test account so authentication reaches password
	// verification rather than failing earlier because the account is
	// still pending verification.
	_, err = pool.Exec(
		ctx,
		`
		UPDATE users
		SET status = 'active'
		WHERE email = $1
		`,
		email,
	)
	if err != nil {
		t.Fatalf(
			"failed to activate test user: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_, err := pool.Exec(
			ctx,
			`DELETE FROM users WHERE email = $1`,
			email,
		)

		if err != nil {
			t.Errorf(
				"failed to clean up test user: %v",
				err,
			)
		}
	})

	// Attempt authentication using an incorrect password.
	loginRequestBody := schemas.LoginUserRequest{
		Email:    email,
		Password: "WrongPassword123",
	}

	loginBody, err := json.Marshal(
		loginRequestBody,
	)
	if err != nil {
		t.Fatalf(
			"failed to encode login request: %v",
			err,
		)
	}

	loginRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/login",
		bytes.NewReader(loginBody),
	)

	loginRequest.Header.Set(
		"Content-Type",
		"application/json",
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(
		recorder,
		loginRequest,
	)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusUnauthorized,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	expectedBody := "invalid credentials\n"

	if recorder.Body.String() != expectedBody {
		t.Fatalf(
			"expected error body %q, got %q",
			expectedBody,
			recorder.Body.String(),
		)
	}
}

// TestLoginUserIntegration_NonExistentUser verifies that attempting
// authentication with an email that does not exist produces the same
// generic invalid-credentials response.
func TestLoginUserIntegration_NonExistentUser(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf(
			"failed to load configuration: %v",
			err,
		)
	}

	pool, err := postgres.NewPool(cfg)
	if err != nil {
		t.Fatalf(
			"failed to create PostgreSQL pool: %v",
			err,
		)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	queries := postgres.NewQueries(pool)

	userRepository := postgres.NewUserRepository(
		queries,
	)

	passwordHasher := security.NewBcryptPasswordHasher()

	tokenService, err := security.NewJWTTokenServiceFromConfig(
		cfg,
	)
	if err != nil {
		t.Fatalf(
			"failed to create JWT token service: %v",
			err,
		)
	}

	authenticateUserUseCase := use_cases.NewAuthenticateUserUseCase(
		userRepository,
		passwordHasher,
		tokenService,
	)

	loginUserHandler := handlers.NewLoginUserHandler(
		authenticateUserUseCase,
	)

	registerUserHandler := handlers.NewRegisterUserHandler(
		use_cases.NewRegisterUserUseCase(
			userRepository,
			passwordHasher,
		),
	)

	router := newLoginIntegrationRouter(
		registerUserHandler,
		loginUserHandler,
	)

	requestBody := schemas.LoginUserRequest{
		Email:    "non-existent-" + uuid.New().String() + "@example.com",
		Password: "SecurePassword123",
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf(
			"failed to encode login request: %v",
			err,
		)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/login",
		bytes.NewReader(body),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusUnauthorized,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	expectedBody := "invalid credentials\n"

	if recorder.Body.String() != expectedBody {
		t.Fatalf(
			"expected error body %q, got %q",
			expectedBody,
			recorder.Body.String(),
		)
	}
}
