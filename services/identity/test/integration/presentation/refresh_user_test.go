package presentation_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/persistence/postgres"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/security"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/schemas"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/shared/config"
)

// newRefreshIntegrationRouter creates the real Identity HTTP router for
// refresh-token integration tests.
//
// The refresh handler and refresh use case are real.
//
// Registration/login dependencies exist only because the router requires
// them. They are not part of the refresh-token flow being tested.
func newRefreshIntegrationRouter(
	refreshUserHandler *handlers.RefreshUserHandler,
) http.Handler {
	registerUserHandler := handlers.NewRegisterUserHandler(
		&mockIntegrationRegisterUserService{},
	)

	loginUserHandler := handlers.NewLoginUserHandler(
		&mockIntegrationAuthenticateUserService{},
	)

	meHandler := handlers.NewMeHandler()

	authenticationMiddleware := middleware.NewAuthenticationMiddleware(
		&mockIntegrationTokenService{},
	)

	// Create the logout handler required by the router.
	// This test does not exercise the logout endpoint yet.
	logoutUserHandler := handlers.NewLogoutUserHandler(
		&mockRouterLogoutUserService{},
	)

	router := presentation.NewRouter(
		registerUserHandler,
		loginUserHandler,
		refreshUserHandler,
		meHandler,
		authenticationMiddleware,
		logoutUserHandler,
	)
	return router
}

// mockIntegrationRegisterUserService exists only to satisfy the router
// dependency. Registration is not part of this test's HTTP flow.
type mockIntegrationRegisterUserService struct{}

func (m *mockIntegrationRegisterUserService) Execute(
	ctx context.Context,
	command dto.RegisterUserCommand,
) (dto.RegisterUserResult, error) {
	return dto.RegisterUserResult{}, nil
}

// TestRefreshUserIntegration verifies the complete refresh-token flow.
//
// The test uses real:
//
//	HTTP request
//	    ↓
//	RefreshUserHandler
//	    ↓
//	RefreshUserUseCase
//	    ↓
//	RefreshTokenService
//	    ↓
//	PostgreSQL RefreshTokenRepository
//	    ↓
//	UserRepository
//	    ↓
//	JWT TokenService
//	    ↓
//	HTTP response
//
// It also verifies refresh-token rotation:
//   - the old refresh token is revoked
//   - a new refresh-token session is persisted
//   - the replacement token belongs to the current user
//   - the new access token is a valid JWT
func TestRefreshUserIntegration(t *testing.T) {
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
	// Build real persistence dependencies
	// ---------------------------------------------------------

	queries := postgres.NewQueries(pool)

	userRepository := postgres.NewUserRepository(
		queries,
	)

	refreshTokenRepository := postgres.NewRefreshTokenRepository(
		pool,
		queries,
	)

	// ---------------------------------------------------------
	// Build real security services
	// ---------------------------------------------------------

	refreshTokenService := security.NewRefreshTokenService()

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
	// Build real refresh use case
	// ---------------------------------------------------------

	refreshUserUseCase := use_cases.NewRefreshUserUseCase(
		refreshTokenRepository,
		userRepository,
		refreshTokenService,
		tokenService,
	)

	// ---------------------------------------------------------
	// Build real refresh handler
	// ---------------------------------------------------------

	refreshUserHandler := handlers.NewRefreshUserHandler(
		refreshUserUseCase,
	)

	// ---------------------------------------------------------
	// Build real Identity router
	// ---------------------------------------------------------

	router := newRefreshIntegrationRouter(
		refreshUserHandler,
	)

	// ---------------------------------------------------------
	// Create a real active test user
	// ---------------------------------------------------------

	email := "refresh-" + uuid.New().String() + "@example.com"

	fullName := "Refresh Integration User"

	// Use the registration use case directly to create the real user.
	//
	// The refresh flow itself starts after the user already exists, so there
	// is no reason to exercise the registration HTTP endpoint again here.
	passwordHasher := security.NewBcryptPasswordHasher()

	registerUserUseCase := use_cases.NewRegisterUserUseCase(
		userRepository,
		passwordHasher,
	)

	registerResult, err := registerUserUseCase.Execute(
		ctx,
		dto.RegisterUserCommand{
			FullName: fullName,
			Email:    email,
			Password: "SecurePassword123",
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to create test user: %v",
			err,
		)
	}

	// The registration use case intentionally creates the account in its
	// initial verification state.
	//
	// Activate the account directly because this test is concerned with
	// refresh authentication rather than account verification.
	_, err = pool.Exec(
		ctx,
		`
		UPDATE users
		SET status = 'active'
		WHERE id = $1
		`,
		registerResult.ID,
	)
	if err != nil {
		t.Fatalf(
			"failed to activate test user: %v",
			err,
		)
	}

	// ---------------------------------------------------------
	// Clean up the test user
	// ---------------------------------------------------------

	t.Cleanup(func() {
		_, err := pool.Exec(
			ctx,
			`DELETE FROM users WHERE id = $1`,
			registerResult.ID,
		)

		if err != nil {
			t.Errorf(
				"failed to clean up test user: %v",
				err,
			)
		}
	})

	// ---------------------------------------------------------
	// Generate the initial refresh token
	// ---------------------------------------------------------

	oldRefreshToken, err := refreshTokenService.Generate(
		ctx,
	)
	if err != nil {
		t.Fatalf(
			"failed to generate initial refresh token: %v",
			err,
		)
	}

	oldRefreshTokenHash, err := refreshTokenService.Hash(
		ctx,
		oldRefreshToken,
	)
	if err != nil {
		t.Fatalf(
			"failed to hash initial refresh token: %v",
			err,
		)
	}

	// ---------------------------------------------------------
	// Persist the initial refresh-token session
	// ---------------------------------------------------------

	oldSessionID := uuid.NewString()
	now := time.Now()

	err = refreshTokenRepository.Create(
		ctx,
		ports.RefreshTokenRecord{
			ID:        oldSessionID,
			UserID:    registerResult.ID,
			TokenHash: oldRefreshTokenHash,
			ExpiresAt: now.Add(30 * 24 * time.Hour),
			CreatedAt: now,
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to persist initial refresh-token session: %v",
			err,
		)
	}

	// ---------------------------------------------------------
	// Build refresh request
	// ---------------------------------------------------------

	requestBody := schemas.RefreshTokenRequest{
		RefreshToken: oldRefreshToken,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf(
			"failed to encode refresh request: %v",
			err,
		)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/refresh",
		bytes.NewReader(body),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	recorder := httptest.NewRecorder()

	// ---------------------------------------------------------
	// Execute real refresh endpoint
	// ---------------------------------------------------------

	router.ServeHTTP(
		recorder,
		request,
	)

	// ---------------------------------------------------------
	// Verify HTTP response
	// ---------------------------------------------------------

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected refresh status %d, got %d, body: %s",
			http.StatusOK,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response schemas.RefreshTokenResponse

	if err := json.NewDecoder(
		recorder.Body,
	).Decode(&response); err != nil {
		t.Fatalf(
			"failed to decode refresh response: %v",
			err,
		)
	}

	// ---------------------------------------------------------
	// Verify replacement tokens were returned
	// ---------------------------------------------------------

	if response.AccessToken == "" {
		t.Fatal(
			"expected new access token in refresh response",
		)
	}

	if response.RefreshToken == "" {
		t.Fatal(
			"expected new refresh token in refresh response",
		)
	}

	if response.RefreshToken == oldRefreshToken {
		t.Fatal(
			"expected refresh-token rotation to produce a new refresh token",
		)
	}

	// ---------------------------------------------------------
	// Verify new access token cryptographically
	// ---------------------------------------------------------

	identity, err := tokenService.ValidateAccessToken(
		ctx,
		response.AccessToken,
	)
	if err != nil {
		t.Fatalf(
			"expected returned access token to be valid: %v",
			err,
		)
	}

	if identity.UserID != registerResult.ID {
		t.Fatalf(
			"expected access token user ID %q, got %q",
			registerResult.ID,
			identity.UserID,
		)
	}

	if identity.Role != "user" {
		t.Fatalf(
			"expected access token role %q, got %q",
			"user",
			identity.Role,
		)
	}

	// ---------------------------------------------------------
	// Hash the replacement refresh token
	// ---------------------------------------------------------

	newRefreshTokenHash, err := refreshTokenService.Hash(
		ctx,
		response.RefreshToken,
	)
	if err != nil {
		t.Fatalf(
			"failed to hash replacement refresh token: %v",
			err,
		)
	}

	// ---------------------------------------------------------
	// Verify old session was revoked
	// ---------------------------------------------------------

	oldRecord, err := refreshTokenRepository.FindByTokenHash(
		ctx,
		oldRefreshTokenHash,
	)
	if err != nil {
		t.Fatalf(
			"failed to retrieve old refresh-token session: %v",
			err,
		)
	}

	if oldRecord == nil {
		t.Fatal(
			"expected old refresh-token session to remain persisted",
		)
	}

	if oldRecord.RevokedAt == nil {
		t.Fatal(
			"expected old refresh-token session to be revoked",
		)
	}

	// ---------------------------------------------------------
	// Verify replacement session was persisted
	// ---------------------------------------------------------

	newRecord, err := refreshTokenRepository.FindByTokenHash(
		ctx,
		newRefreshTokenHash,
	)
	if err != nil {
		t.Fatalf(
			"failed to retrieve replacement refresh-token session: %v",
			err,
		)
	}

	if newRecord == nil {
		t.Fatal(
			"expected replacement refresh-token session to be persisted",
		)
	}

	if newRecord.RevokedAt != nil {
		t.Fatal(
			"expected replacement refresh-token session to be active",
		)
	}

	if newRecord.UserID != registerResult.ID {
		t.Fatalf(
			"expected replacement session to belong to user %q, got %q",
			registerResult.ID,
			newRecord.UserID,
		)
	}

	if newRecord.TokenHash != newRefreshTokenHash {
		t.Fatal(
			"expected persisted replacement token hash to match supplied token",
		)
	}

	if !newRecord.ExpiresAt.After(time.Now()) {
		t.Fatal(
			"expected replacement refresh token to expire in the future",
		)
	}

	if newRecord.ID == oldRecord.ID {
		t.Fatal(
			"expected replacement refresh-token session to have a new session ID",
		)
	}

	// ---------------------------------------------------------
	// Verify old refresh token cannot be reused
	// ---------------------------------------------------------

	replayRequestBody := schemas.RefreshTokenRequest{
		RefreshToken: oldRefreshToken,
	}

	replayBody, err := json.Marshal(replayRequestBody)
	if err != nil {
		t.Fatalf(
			"failed to encode replay refresh request: %v",
			err,
		)
	}

	replayRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/refresh",
		bytes.NewReader(replayBody),
	)

	replayRequest.Header.Set(
		"Content-Type",
		"application/json",
	)

	replayRecorder := httptest.NewRecorder()

	// The old refresh token was already rotated and revoked.
	//
	// Reusing it must therefore fail authentication.
	router.ServeHTTP(
		replayRecorder,
		replayRequest,
	)

	if replayRecorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected replayed refresh token status %d, got %d, body: %s",
			http.StatusUnauthorized,
			replayRecorder.Code,
			replayRecorder.Body.String(),
		)
	}
}

func TestRefreshUserIntegration_InvalidToken(t *testing.T) {

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
	refreshTokenRepository := postgres.NewRefreshTokenRepository(pool, queries)
	refreshTokenService := security.NewRefreshTokenService()

	tokenService, err := security.NewJWTTokenServiceFromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create JWT token service: %v", err)
	}

	refreshUserUseCase := use_cases.NewRefreshUserUseCase(
		refreshTokenRepository,
		userRepository,
		refreshTokenService,
		tokenService,
	)

	refreshUserHandler := handlers.NewRefreshUserHandler(
		refreshUserUseCase,
	)

	router := newRefreshIntegrationRouter(refreshUserHandler)

	requestBody := schemas.RefreshTokenRequest{
		RefreshToken: "this-is-not-a-valid-refresh-token",
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/refresh",
		bytes.NewReader(body),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusUnauthorized,
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func TestRefreshUserIntegration_ExpiredToken(t *testing.T) {
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
	refreshTokenRepository := postgres.NewRefreshTokenRepository(pool, queries)
	refreshTokenService := security.NewRefreshTokenService()

	tokenService, err := security.NewJWTTokenServiceFromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create JWT token service: %v", err)
	}

	refreshUserUseCase := use_cases.NewRefreshUserUseCase(
		refreshTokenRepository,
		userRepository,
		refreshTokenService,
		tokenService,
	)

	refreshUserHandler := handlers.NewRefreshUserHandler(
		refreshUserUseCase,
	)

	router := newRefreshIntegrationRouter(refreshUserHandler)

	// Create a real user.
	email := "expired-refresh-" + uuid.New().String() + "@example.com"

	passwordHasher := security.NewBcryptPasswordHasher()

	registerUserUseCase := use_cases.NewRegisterUserUseCase(
		userRepository,
		passwordHasher,
	)

	registerResult, err := registerUserUseCase.Execute(
		ctx,
		dto.RegisterUserCommand{
			FullName: "Expired Refresh Integration User",
			Email:    email,
			Password: "SecurePassword123",
		},
	)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	t.Cleanup(func() {
		_, err := pool.Exec(
			ctx,
			`DELETE FROM users WHERE id = $1`,
			registerResult.ID,
		)

		if err != nil {
			t.Errorf(
				"failed to clean up test user: %v",
				err,
			)
		}
	})

	_, err = pool.Exec(
		ctx,
		`
		UPDATE users
		SET status = 'active'
		WHERE id = $1
		`,
		registerResult.ID,
	)
	if err != nil {
		t.Fatalf("failed to activate test user: %v", err)
	}

	// Generate and hash the refresh token.
	refreshToken, err := refreshTokenService.Generate(ctx)
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}

	refreshTokenHash, err := refreshTokenService.Hash(
		ctx,
		refreshToken,
	)
	if err != nil {
		t.Fatalf("failed to hash refresh token: %v", err)
	}

	// Persist an already-expired session.
	err = refreshTokenRepository.Create(
		ctx,
		ports.RefreshTokenRecord{
			ID:        uuid.NewString(),
			UserID:    registerResult.ID,
			TokenHash: refreshTokenHash,
			ExpiresAt: time.Now().Add(-1 * time.Minute),
			CreatedAt: time.Now().Add(-2 * time.Minute),
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to persist expired refresh-token session: %v",
			err,
		)
	}

	requestBody := schemas.RefreshTokenRequest{
		RefreshToken: refreshToken,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/refresh",
		bytes.NewReader(body),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusUnauthorized,
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func TestRefreshUserIntegration_RevokedToken(t *testing.T) {
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
	refreshTokenRepository := postgres.NewRefreshTokenRepository(pool, queries)
	refreshTokenService := security.NewRefreshTokenService()

	tokenService, err := security.NewJWTTokenServiceFromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create JWT token service: %v", err)
	}

	refreshUserUseCase := use_cases.NewRefreshUserUseCase(
		refreshTokenRepository,
		userRepository,
		refreshTokenService,
		tokenService,
	)

	refreshUserHandler := handlers.NewRefreshUserHandler(
		refreshUserUseCase,
	)

	router := newRefreshIntegrationRouter(refreshUserHandler)

	// Create a real user.
	email := "revoked-refresh-" + uuid.New().String() + "@example.com"

	passwordHasher := security.NewBcryptPasswordHasher()

	registerUserUseCase := use_cases.NewRegisterUserUseCase(
		userRepository,
		passwordHasher,
	)

	registerResult, err := registerUserUseCase.Execute(
		ctx,
		dto.RegisterUserCommand{
			FullName: "Revoked Refresh Integration User",
			Email:    email,
			Password: "SecurePassword123",
		},
	)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	t.Cleanup(func() {
		_, err := pool.Exec(
			ctx,
			`DELETE FROM users WHERE id = $1`,
			registerResult.ID,
		)

		if err != nil {
			t.Errorf(
				"failed to clean up test user: %v",
				err,
			)
		}
	})

	_, err = pool.Exec(
		ctx,
		`
		UPDATE users
		SET status = 'active'
		WHERE id = $1
		`,
		registerResult.ID,
	)
	if err != nil {
		t.Fatalf("failed to activate test user: %v", err)
	}

	// Generate and hash the refresh token.
	refreshToken, err := refreshTokenService.Generate(ctx)
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}

	refreshTokenHash, err := refreshTokenService.Hash(
		ctx,
		refreshToken,
	)
	if err != nil {
		t.Fatalf("failed to hash refresh token: %v", err)
	}

	sessionID := uuid.NewString()

	// Persist an active session first.
	err = refreshTokenRepository.Create(
		ctx,
		ports.RefreshTokenRecord{
			ID:        sessionID,
			UserID:    registerResult.ID,
			TokenHash: refreshTokenHash,
			ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
			CreatedAt: time.Now(),
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to persist refresh-token session: %v",
			err,
		)
	}

	// Explicitly revoke the session.
	err = refreshTokenRepository.Revoke(
		ctx,
		sessionID,
		time.Now(),
	)
	if err != nil {
		t.Fatalf(
			"failed to revoke refresh-token session: %v",
			err,
		)
	}

	requestBody := schemas.RefreshTokenRequest{
		RefreshToken: refreshToken,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/refresh",
		bytes.NewReader(body),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusUnauthorized,
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func TestRefreshUserIntegration_InactiveUser(t *testing.T) {
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
	refreshTokenRepository := postgres.NewRefreshTokenRepository(pool, queries)
	refreshTokenService := security.NewRefreshTokenService()

	tokenService, err := security.NewJWTTokenServiceFromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create JWT token service: %v", err)
	}

	refreshUserUseCase := use_cases.NewRefreshUserUseCase(
		refreshTokenRepository,
		userRepository,
		refreshTokenService,
		tokenService,
	)

	refreshUserHandler := handlers.NewRefreshUserHandler(
		refreshUserUseCase,
	)

	router := newRefreshIntegrationRouter(refreshUserHandler)

	// ---------------------------------------------------------
	// Create a real test user
	// ---------------------------------------------------------

	email := "inactive-refresh-" + uuid.New().String() + "@example.com"

	passwordHasher := security.NewBcryptPasswordHasher()

	registerUserUseCase := use_cases.NewRegisterUserUseCase(
		userRepository,
		passwordHasher,
	)

	registerResult, err := registerUserUseCase.Execute(
		ctx,
		dto.RegisterUserCommand{
			FullName: "Inactive Refresh Integration User",
			Email:    email,
			Password: "SecurePassword123",
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to create test user: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_, err := pool.Exec(
			ctx,
			`DELETE FROM users WHERE id = $1`,
			registerResult.ID,
		)

		if err != nil {
			t.Errorf(
				"failed to clean up test user: %v",
				err,
			)
		}
	})

	// ---------------------------------------------------------
	// Set the account to inactive
	// ---------------------------------------------------------

	_, err = pool.Exec(
		ctx,
		`
		UPDATE users
		SET status = 'inactive'
		WHERE id = $1
		`,
		registerResult.ID,
	)
	if err != nil {
		t.Fatalf(
			"failed to deactivate test user: %v",
			err,
		)
	}

	// ---------------------------------------------------------
	// Generate refresh token
	// ---------------------------------------------------------

	refreshToken, err := refreshTokenService.Generate(ctx)
	if err != nil {
		t.Fatalf(
			"failed to generate refresh token: %v",
			err,
		)
	}

	refreshTokenHash, err := refreshTokenService.Hash(
		ctx,
		refreshToken,
	)
	if err != nil {
		t.Fatalf(
			"failed to hash refresh token: %v",
			err,
		)
	}

	// ---------------------------------------------------------
	// Persist valid refresh-token session
	// ---------------------------------------------------------

	err = refreshTokenRepository.Create(
		ctx,
		ports.RefreshTokenRecord{
			ID:        uuid.NewString(),
			UserID:    registerResult.ID,
			TokenHash: refreshTokenHash,
			ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
			CreatedAt: time.Now(),
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to persist refresh-token session: %v",
			err,
		)
	}

	// ---------------------------------------------------------
	// Build HTTP request
	// ---------------------------------------------------------

	requestBody := schemas.RefreshTokenRequest{
		RefreshToken: refreshToken,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf(
			"failed to encode request: %v",
			err,
		)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/refresh",
		bytes.NewReader(body),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	recorder := httptest.NewRecorder()

	// ---------------------------------------------------------
	// Execute refresh endpoint
	// ---------------------------------------------------------

	router.ServeHTTP(
		recorder,
		request,
	)

	// ---------------------------------------------------------
	// Verify inactive account is rejected
	// ---------------------------------------------------------

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusUnauthorized,
			recorder.Code,
			recorder.Body.String(),
		)
	}
}
