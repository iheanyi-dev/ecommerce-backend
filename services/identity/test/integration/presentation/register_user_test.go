package presentation_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/schemas"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/shared/config"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
)

// mockIntegrationAuthenticateUserService is a lightweight authentication
// service used only because the Identity router now contains both
// registration and login routes.
//
// These registration integration tests are not testing authentication,
// so we deliberately do not connect the login handler to the real
// authentication use case here.
type mockIntegrationAuthenticateUserService struct{}

func (m *mockIntegrationAuthenticateUserService) Authenticate(
	ctx context.Context,
	command dto.LoginUserCommand,
) (dto.LoginUserResult, error) {
	return dto.LoginUserResult{}, errors.New(
		"authentication is not part of this integration test",
	)
}

type mockIntegrationTokenService struct{}

func (m *mockIntegrationTokenService) GenerateAccessToken(
	ctx context.Context,
	userID user.UserID,
	role user.Role,
) (string, error) {
	return "integration-test-token", nil
}

func (m *mockIntegrationTokenService) ValidateAccessToken(
	ctx context.Context,
	token string,
) (ports.AuthenticatedIdentity, error) {
	return ports.AuthenticatedIdentity{}, errors.New(
		"authentication is not part of this integration test",
	)
}

// newRegistrationIntegrationRouter creates the real Identity HTTP router
// for registration integration tests.
//
// The registration handler is real and connected to PostgreSQL, sqlc,
// bcrypt, and the registration use case.
//
// The login handler only exists to satisfy the router's dependency because
// authentication is tested separately.
func newRegistrationIntegrationRouter(
	registerUserHandler *handlers.RegisterUserHandler,
) http.Handler {
	loginUserHandler := handlers.NewLoginUserHandler(
		&mockIntegrationAuthenticateUserService{},
	)

	meHandler := handlers.NewMeHandler()

	authenticationMiddleware := middleware.NewAuthenticationMiddleware(
		&mockIntegrationTokenService{},
	)

	return presentation.NewRouter(
		registerUserHandler,
		loginUserHandler,
		meHandler,
		authenticationMiddleware,
	)
}

func TestRegisterUserIntegration(t *testing.T) {
	ctx := context.Background()

	// Load the same configuration used by the Identity service.
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load configuration: %v", err)
	}

	// Create a real PostgreSQL connection pool.
	pool, err := postgres.NewPool(cfg)
	if err != nil {
		t.Fatalf("failed to create PostgreSQL pool: %v", err)
	}

	// Build the real SQLC query executor.
	queries := postgres.NewQueries(pool)

	// Build the real PostgreSQL repository.
	userRepository := postgres.NewUserRepository(queries)

	// Build the real password hasher.
	passwordHasher := security.NewBcryptPasswordHasher()

	// Build the real registration use case.
	registerUserUseCase := use_cases.NewRegisterUserUseCase(
		userRepository,
		passwordHasher,
	)

	// Build the real HTTP handler.
	registerUserHandler := handlers.NewRegisterUserHandler(
		registerUserUseCase,
	)

	// Build the real HTTP router.
	//
	// The registration handler is real. The login handler is a lightweight
	// test dependency because this integration test is specifically testing
	// registration.
	router := newRegistrationIntegrationRouter(
		registerUserHandler,
	)

	// Use a unique email so repeated test runs do not collide with the
	// database's unique email constraint.
	email := fmt.Sprintf(
		"registration-%s@example.com",
		uuid.New().String(),
	)

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

		pool.Close()
	})

	// Remove the user if it already exists from a previous test execution.
	_, err = pool.Exec(
		ctx,
		`DELETE FROM users WHERE email = $1`,
		email,
	)
	if err != nil {
		t.Fatalf("failed to clean test user: %v", err)
	}

	requestBody := schemas.RegisterUserRequest{
		FullName: "John Doe",
		Email:    email,
		Password: "SecurePassword123",
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("failed to encode request body: %v", err)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/register",
		bytes.NewReader(body),
	)

	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusCreated,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	// Verify that the user was actually persisted.
	var (
		storedEmail        string
		storedPasswordHash string
	)

	err = pool.QueryRow(
		ctx,
		`
		SELECT email, password_hash
		FROM users
		WHERE email = $1
		`,
		email,
	).Scan(
		&storedEmail,
		&storedPasswordHash,
	)

	if err != nil {
		t.Fatalf(
			"failed to retrieve registered user: %v",
			err,
		)
	}

	if storedEmail != email {
		t.Fatalf(
			"expected stored email %q, got %q",
			email,
			storedEmail,
		)
	}

	if storedPasswordHash == requestBody.Password {
		t.Fatal("password was stored as plaintext")
	}

	if storedPasswordHash == "" {
		t.Fatal("expected password hash to be stored")
	}
}

func TestRegisterUserIntegration_DuplicateEmail(t *testing.T) {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load configuration: %v", err)
	}

	pool, err := postgres.NewPool(cfg)
	if err != nil {
		t.Fatalf("failed to create PostgreSQL pool: %v", err)
	}

	queries := postgres.NewQueries(pool)

	userRepository := postgres.NewUserRepository(queries)

	passwordHasher := security.NewBcryptPasswordHasher()

	registerUserUseCase := use_cases.NewRegisterUserUseCase(
		userRepository,
		passwordHasher,
	)

	registerUserHandler := handlers.NewRegisterUserHandler(
		registerUserUseCase,
	)

	router := newRegistrationIntegrationRouter(
		registerUserHandler,
	)

	email := fmt.Sprintf(
		"duplicate-%s@example.com",
		uuid.New().String(),
	)

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

		pool.Close()
	})

	// ---------------------------------------------------------
	// First registration
	// ---------------------------------------------------------

	firstRequestBody := schemas.RegisterUserRequest{
		FullName: "John Doe",
		Email:    email,
		Password: "SecurePassword123",
	}

	firstBody, err := json.Marshal(firstRequestBody)
	if err != nil {
		t.Fatalf("failed to encode first request: %v", err)
	}

	firstRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/register",
		bytes.NewReader(firstBody),
	)

	firstRequest.Header.Set(
		"Content-Type",
		"application/json",
	)

	firstRecorder := httptest.NewRecorder()

	router.ServeHTTP(
		firstRecorder,
		firstRequest,
	)

	if firstRecorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected first registration status %d, got %d, body: %s",
			http.StatusCreated,
			firstRecorder.Code,
			firstRecorder.Body.String(),
		)
	}

	// ---------------------------------------------------------
	// Second registration using the same email
	// ---------------------------------------------------------

	secondRequestBody := schemas.RegisterUserRequest{
		FullName: "Jane Doe",
		Email:    email,
		Password: "AnotherPassword123",
	}

	secondBody, err := json.Marshal(secondRequestBody)
	if err != nil {
		t.Fatalf("failed to encode second request: %v", err)
	}

	secondRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/register",
		bytes.NewReader(secondBody),
	)

	secondRequest.Header.Set(
		"Content-Type",
		"application/json",
	)

	secondRecorder := httptest.NewRecorder()

	router.ServeHTTP(
		secondRecorder,
		secondRequest,
	)

	// The application layer should detect that the email already exists
	// and the presentation layer should translate that error into 409.
	if secondRecorder.Code != http.StatusConflict {
		t.Fatalf(
			"expected duplicate registration status %d, got %d, body: %s",
			http.StatusConflict,
			secondRecorder.Code,
			secondRecorder.Body.String(),
		)
	}

	var response map[string]string

	if err := json.NewDecoder(
		secondRecorder.Body,
	).Decode(&response); err != nil {
		t.Fatalf(
			"failed to decode duplicate registration response: %v",
			err,
		)
	}

	if response["error"] != "email already exists" {
		t.Fatalf(
			"expected error %q, got %q",
			"email already exists",
			response["error"],
		)
	}
}

func TestRegisterUserIntegration_InvalidEmail(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load configuration: %v", err)
	}

	pool, err := postgres.NewPool(cfg)
	if err != nil {
		t.Fatalf("failed to create PostgreSQL pool: %v", err)
	}

	queries := postgres.NewQueries(pool)
	userRepository := postgres.NewUserRepository(queries)
	passwordHasher := security.NewBcryptPasswordHasher()

	registerUserUseCase := use_cases.NewRegisterUserUseCase(
		userRepository,
		passwordHasher,
	)

	registerUserHandler := handlers.NewRegisterUserHandler(
		registerUserUseCase,
	)

	router := newRegistrationIntegrationRouter(
		registerUserHandler,
	)

	t.Cleanup(func() {
		pool.Close()
	})

	requestBody := schemas.RegisterUserRequest{
		FullName: "John Doe",
		Email:    "not-an-email",
		Password: "SecurePassword123",
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("failed to encode request body: %v", err)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/register",
		bytes.NewReader(body),
	)

	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusBadRequest,
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func TestRegisterUserIntegration_InvalidFullName(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load configuration: %v", err)
	}

	pool, err := postgres.NewPool(cfg)
	if err != nil {
		t.Fatalf("failed to create PostgreSQL pool: %v", err)
	}

	queries := postgres.NewQueries(pool)
	userRepository := postgres.NewUserRepository(queries)
	passwordHasher := security.NewBcryptPasswordHasher()

	registerUserUseCase := use_cases.NewRegisterUserUseCase(
		userRepository,
		passwordHasher,
	)

	registerUserHandler := handlers.NewRegisterUserHandler(
		registerUserUseCase,
	)

	router := newRegistrationIntegrationRouter(
		registerUserHandler,
	)

	t.Cleanup(func() {
		pool.Close()
	})

	requestBody := schemas.RegisterUserRequest{
		FullName: "",
		Email:    "valid@example.com",
		Password: "SecurePassword123",
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("failed to encode request body: %v", err)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/register",
		bytes.NewReader(body),
	)

	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusBadRequest,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response map[string]string

	if err := json.NewDecoder(
		recorder.Body,
	).Decode(&response); err != nil {
		t.Fatalf(
			"failed to decode error response: %v",
			err,
		)
	}

	if response["error"] != "invalid full name" {
		t.Fatalf(
			"expected error %q, got %q",
			"invalid full name",
			response["error"],
		)
	}
}

func TestRegisterUserIntegration_InvalidJSON(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load configuration: %v", err)
	}

	pool, err := postgres.NewPool(cfg)
	if err != nil {
		t.Fatalf("failed to create PostgreSQL pool: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	queries := postgres.NewQueries(pool)
	userRepository := postgres.NewUserRepository(queries)
	passwordHasher := security.NewBcryptPasswordHasher()

	registerUserUseCase := use_cases.NewRegisterUserUseCase(
		userRepository,
		passwordHasher,
	)

	registerUserHandler := handlers.NewRegisterUserHandler(
		registerUserUseCase,
	)

	router := newRegistrationIntegrationRouter(
		registerUserHandler,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/register",
		bytes.NewBufferString(`{"full_name":"John Doe","email":`),
	)

	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusBadRequest,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response map[string]string

	if err := json.NewDecoder(
		recorder.Body,
	).Decode(&response); err != nil {
		t.Fatalf(
			"failed to decode error response: %v",
			err,
		)
	}

	if response["error"] != "invalid request body" {
		t.Fatalf(
			"expected error %q, got %q",
			"invalid request body",
			response["error"],
		)
	}
}

func TestRegisterUserIntegration_MethodNotAllowed(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load configuration: %v", err)
	}

	pool, err := postgres.NewPool(cfg)
	if err != nil {
		t.Fatalf("failed to create PostgreSQL pool: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	queries := postgres.NewQueries(pool)
	userRepository := postgres.NewUserRepository(queries)
	passwordHasher := security.NewBcryptPasswordHasher()

	registerUserUseCase := use_cases.NewRegisterUserUseCase(
		userRepository,
		passwordHasher,
	)

	registerUserHandler := handlers.NewRegisterUserHandler(
		registerUserUseCase,
	)

	router := newRegistrationIntegrationRouter(
		registerUserHandler,
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/register",
		nil,
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusMethodNotAllowed,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response map[string]string

	if err := json.NewDecoder(
		recorder.Body,
	).Decode(&response); err != nil {
		t.Fatalf(
			"failed to decode error response: %v",
			err,
		)
	}

	if response["error"] != "method not allowed" {
		t.Fatalf(
			"expected error %q, got %q",
			"method not allowed",
			response["error"],
		)
	}
}

func TestRegisterUserIntegration_Route(t *testing.T) {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load configuration: %v", err)
	}

	pool, err := postgres.NewPool(cfg)
	if err != nil {
		t.Fatalf("failed to create PostgreSQL pool: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	queries := postgres.NewQueries(pool)

	userRepository := postgres.NewUserRepository(queries)

	passwordHasher := security.NewBcryptPasswordHasher()

	registerUserUseCase := use_cases.NewRegisterUserUseCase(
		userRepository,
		passwordHasher,
	)

	registerUserHandler := handlers.NewRegisterUserHandler(
		registerUserUseCase,
	)

	router := newRegistrationIntegrationRouter(
		registerUserHandler,
	)

	email := fmt.Sprintf(
		"route-%s@example.com",
		uuid.New().String(),
	)

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

	body := schemas.RegisterUserRequest{
		FullName: "Route Test User",
		Email:    email,
		Password: "SecurePassword123",
	}

	requestBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/register",
		bytes.NewReader(requestBody),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusCreated,
			recorder.Code,
			recorder.Body.String(),
		)
	}
}