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
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/persistence/postgres"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/security"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/schemas"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/shared/config"
)

// newLogoutIntegrationRouter creates the real Identity HTTP router
// with the real logout handler, authentication middleware, and
// application dependencies.
//
// The other handlers are supplied only because the router requires
// them. They are not exercised by these logout integration tests.
func newLogoutIntegrationRouter(
	logoutUserHandler *handlers.LogoutUserHandler,
	tokenService ports.TokenService,
) http.Handler {
	registerUserHandler := handlers.NewRegisterUserHandler(
		&mockIntegrationRegisterUserService{},
	)

	loginUserHandler := handlers.NewLoginUserHandler(
		&mockIntegrationAuthenticateUserService{},
	)

	refreshUserHandler := handlers.NewRefreshUserHandler(
		&mockIntegrationRefreshUserService{},
	)

	meHandler := handlers.NewMeHandler()

	authenticationMiddleware := middleware.NewAuthenticationMiddleware(
		tokenService,
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

// TestLogoutUserIntegration verifies the complete authenticated
// logout flow using the real HTTP handler, authentication middleware,
// logout use case, refresh-token service, and PostgreSQL repository.
//
// The important business rule is that logout revokes only the
// refresh-token session represented by the supplied token.
func TestLogoutUserIntegration(t *testing.T) {
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

	refreshTokenRepository := postgres.NewRefreshTokenRepository(
		pool,
		queries,
	)

	refreshTokenService := security.NewRefreshTokenService()

	tokenService, err := security.NewJWTTokenServiceFromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create JWT token service: %v", err)
	}

	logoutUserUseCase := use_cases.NewLogoutUserUseCase(
		refreshTokenRepository,
		refreshTokenService,
	)

	logoutUserHandler := handlers.NewLogoutUserHandler(
		logoutUserUseCase,
	)

	router := newLogoutIntegrationRouter(
		logoutUserHandler,
		tokenService,
	)

	// Create a real user so the refresh-token session has a valid
	// foreign-key relationship with the users table.
	email := "logout-" + uuid.New().String() + "@example.com"

	passwordHasher := security.NewBcryptPasswordHasher()

	registerUserUseCase := use_cases.NewRegisterUserUseCase(
		userRepository,
		passwordHasher,
	)

	registerResult, err := registerUserUseCase.Execute(
		ctx,
		dto.RegisterUserCommand{
			FullName: "Logout Integration User",
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

	// Activate the account so it represents a valid authenticated user.
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

	// Convert the persisted string ID into the domain-specific UserID.
	testUserID, err := user.UserIDFromString(registerResult.ID)
	if err != nil {
		t.Fatalf(
			"failed to create domain user ID: %v",
			err,
		)
	}

	// Generate a real JWT access token for the test user.
	//
	// The authentication middleware validates this token before
	// allowing the logout handler to execute.
	accessToken, err := tokenService.GenerateAccessToken(
		ctx,
		testUserID,
		user.Role("user"),
	)
	if err != nil {
		t.Fatalf(
			"failed to generate access token: %v",
			err,
		)
	}

	// Generate the refresh token representing this device/session.
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

	sessionID := uuid.NewString()
	now := time.Now()

	err = refreshTokenRepository.Create(
		ctx,
		ports.RefreshTokenRecord{
			ID:        sessionID,
			UserID:    registerResult.ID,
			TokenHash: refreshTokenHash,
			ExpiresAt: now.Add(30 * 24 * time.Hour),
			CreatedAt: now,
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to persist refresh-token session: %v",
			err,
		)
	}

	// Build the logout request using the real API schema.
	requestBody := schemas.LogoutRequest{
		RefreshToken: refreshToken,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf(
			"failed to encode logout request: %v",
			err,
		)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/logout",
		bytes.NewReader(body),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	request.Header.Set(
		"Authorization",
		"Bearer "+accessToken,
	)

	recorder := httptest.NewRecorder()

	// Execute the complete authenticated logout flow.
	router.ServeHTTP(
		recorder,
		request,
	)

	// Successful logout returns 204 No Content.
	if recorder.Code != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusNoContent,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	// Retrieve the session from PostgreSQL after logout.
	record, err := refreshTokenRepository.FindByTokenHash(
		ctx,
		refreshTokenHash,
	)
	if err != nil {
		t.Fatalf(
			"failed to retrieve refresh-token session: %v",
			err,
		)
	}

	if record == nil {
		t.Fatal(
			"expected refresh-token session to remain persisted",
		)
	}

	// Logout revokes the session instead of deleting it.
	if record.RevokedAt == nil {
		t.Fatal(
			"expected refresh-token session to be revoked",
		)
	}

	if record.ID != sessionID {
		t.Fatalf(
			"expected revoked session ID %q, got %q",
			sessionID,
			record.ID,
		)
	}
}

// TestLogoutUserIntegration_MissingAccessToken verifies that the
// authentication middleware rejects logout requests without an
// access token before the logout use case is reached.
func TestLogoutUserIntegration_MissingAccessToken(t *testing.T) {
	router, _, _, cleanup := setupLogoutIntegration(t)
	t.Cleanup(cleanup)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/logout",
		bytes.NewReader(
			[]byte(`{"refresh_token":"test-refresh-token"}`),
		),
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
}

// TestLogoutUserIntegration_InvalidAccessToken verifies that an
// invalid access token prevents the logout endpoint from executing.
func TestLogoutUserIntegration_InvalidAccessToken(t *testing.T) {
	router, _, _, cleanup := setupLogoutIntegration(t)
	t.Cleanup(cleanup)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/logout",
		bytes.NewReader(
			[]byte(`{"refresh_token":"test-refresh-token"}`),
		),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	request.Header.Set(
		"Authorization",
		"Bearer definitely-invalid-access-token",
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
}

// TestLogoutUserIntegration_InvalidRefreshToken verifies that a
// valid authenticated request cannot revoke a refresh-token session
// that does not exist.
func TestLogoutUserIntegration_InvalidRefreshToken(t *testing.T) {
	ctx := context.Background()

	router, _, tokenService, cleanup := setupLogoutIntegration(t)
	t.Cleanup(cleanup)

	testUserID, err := user.UserIDFromString(uuid.NewString())
	if err != nil {
		t.Fatalf(
			"failed to create domain user ID: %v",
			err,
		)
	}

	accessToken, err := tokenService.GenerateAccessToken(
		ctx,
		testUserID,
		user.Role("user"),
	)
	if err != nil {
		t.Fatalf(
			"failed to generate access token: %v",
			err,
		)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/logout",
		bytes.NewReader(
			[]byte(`{"refresh_token":"unknown-refresh-token"}`),
		),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	request.Header.Set(
		"Authorization",
		"Bearer "+accessToken,
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
}

// TestLogoutUserIntegration_AlreadyRevokedToken verifies that an
// already-revoked refresh-token session cannot be logged out again.
//
// Unlike the previous version of this test, this test creates a real
// user and a real persisted refresh-token session, revokes it in
// PostgreSQL, and then attempts to use it through the real HTTP API.
func TestLogoutUserIntegration_AlreadyRevokedToken(t *testing.T) {
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

	refreshTokenRepository := postgres.NewRefreshTokenRepository(
		pool,
		queries,
	)

	refreshTokenService := security.NewRefreshTokenService()

	tokenService, err := security.NewJWTTokenServiceFromConfig(cfg)
	if err != nil {
		t.Fatalf(
			"failed to create JWT token service: %v",
			err,
		)
	}

	logoutUserUseCase := use_cases.NewLogoutUserUseCase(
		refreshTokenRepository,
		refreshTokenService,
	)

	logoutUserHandler := handlers.NewLogoutUserHandler(
		logoutUserUseCase,
	)

	router := newLogoutIntegrationRouter(
		logoutUserHandler,
		tokenService,
	)

	// Create a real user for the refresh-token foreign-key relationship.
	email := "revoked-logout-" +
		uuid.New().String() +
		"@example.com"

	registerUserUseCase := use_cases.NewRegisterUserUseCase(
		userRepository,
		security.NewBcryptPasswordHasher(),
	)

	registerResult, err := registerUserUseCase.Execute(
		ctx,
		dto.RegisterUserCommand{
			FullName: "Revoked Logout User",
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

	// Activate the test account.
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

	testUserID, err := user.UserIDFromString(registerResult.ID)
	if err != nil {
		t.Fatalf(
			"failed to create domain user ID: %v",
			err,
		)
	}

	accessToken, err := tokenService.GenerateAccessToken(
		ctx,
		testUserID,
		user.Role("user"),
	)
	if err != nil {
		t.Fatalf(
			"failed to generate access token: %v",
			err,
		)
	}

	// Generate and persist the refresh-token session.
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

	sessionID := uuid.NewString()
	now := time.Now()

	err = refreshTokenRepository.Create(
		ctx,
		ports.RefreshTokenRecord{
			ID:        sessionID,
			UserID:    registerResult.ID,
			TokenHash: refreshTokenHash,
			ExpiresAt: now.Add(30 * 24 * time.Hour),
			CreatedAt: now,
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to persist refresh-token session: %v",
			err,
		)
	}

	// Revoke the session before sending the logout request.
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

	body, err := json.Marshal(
		schemas.LogoutRequest{
			RefreshToken: refreshToken,
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to encode logout request: %v",
			err,
		)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/logout",
		bytes.NewReader(body),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	request.Header.Set(
		"Authorization",
		"Bearer "+accessToken,
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(
		recorder,
		request,
	)

	// An already-revoked refresh token is invalid.
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusUnauthorized,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	// Confirm that the session remains revoked.
	record, err := refreshTokenRepository.FindByTokenHash(
		ctx,
		refreshTokenHash,
	)
	if err != nil {
		t.Fatalf(
			"failed to retrieve revoked session: %v",
			err,
		)
	}

	if record == nil {
		t.Fatal(
			"expected revoked refresh-token session to remain persisted",
		)
	}

	if record.RevokedAt == nil {
		t.Fatal(
			"expected refresh-token session to remain revoked",
		)
	}
}

// TestLogoutUserIntegration_RevokesOnlySpecifiedSession verifies
// the multi-device logout business rule.
//
// Two active refresh-token sessions belong to the same user.
// Logging out with session A must revoke session A while session B
// remains active.
func TestLogoutUserIntegration_RevokesOnlySpecifiedSession(t *testing.T) {
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

	refreshTokenRepository := postgres.NewRefreshTokenRepository(
		pool,
		queries,
	)

	refreshTokenService := security.NewRefreshTokenService()

	tokenService, err := security.NewJWTTokenServiceFromConfig(cfg)
	if err != nil {
		t.Fatalf(
			"failed to create JWT token service: %v",
			err,
		)
	}

	logoutUserUseCase := use_cases.NewLogoutUserUseCase(
		refreshTokenRepository,
		refreshTokenService,
	)

	logoutUserHandler := handlers.NewLogoutUserHandler(
		logoutUserUseCase,
	)

	router := newLogoutIntegrationRouter(
		logoutUserHandler,
		tokenService,
	)

	email := "multi-session-logout-" +
		uuid.New().String() +
		"@example.com"

	registerUserUseCase := use_cases.NewRegisterUserUseCase(
		userRepository,
		security.NewBcryptPasswordHasher(),
	)

	registerResult, err := registerUserUseCase.Execute(
		ctx,
		dto.RegisterUserCommand{
			FullName: "Multi Session Logout User",
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

	// Activate the account.
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

	testUserID, err := user.UserIDFromString(registerResult.ID)
	if err != nil {
		t.Fatalf(
			"failed to create domain user ID: %v",
			err,
		)
	}

	accessToken, err := tokenService.GenerateAccessToken(
		ctx,
		testUserID,
		user.Role("user"),
	)
	if err != nil {
		t.Fatalf(
			"failed to generate access token: %v",
			err,
		)
	}

	// Create two independent refresh-token sessions.
	refreshTokenA, err := refreshTokenService.Generate(ctx)
	if err != nil {
		t.Fatalf(
			"failed to generate refresh token A: %v",
			err,
		)
	}

	refreshTokenB, err := refreshTokenService.Generate(ctx)
	if err != nil {
		t.Fatalf(
			"failed to generate refresh token B: %v",
			err,
		)
	}

	hashA, err := refreshTokenService.Hash(
		ctx,
		refreshTokenA,
	)
	if err != nil {
		t.Fatalf(
			"failed to hash refresh token A: %v",
			err,
		)
	}

	hashB, err := refreshTokenService.Hash(
		ctx,
		refreshTokenB,
	)
	if err != nil {
		t.Fatalf(
			"failed to hash refresh token B: %v",
			err,
		)
	}

	now := time.Now()

	sessionA := uuid.NewString()
	sessionB := uuid.NewString()

	err = refreshTokenRepository.Create(
		ctx,
		ports.RefreshTokenRecord{
			ID:        sessionA,
			UserID:    registerResult.ID,
			TokenHash: hashA,
			ExpiresAt: now.Add(30 * 24 * time.Hour),
			CreatedAt: now,
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to persist session A: %v",
			err,
		)
	}

	err = refreshTokenRepository.Create(
		ctx,
		ports.RefreshTokenRecord{
			ID:        sessionB,
			UserID:    registerResult.ID,
			TokenHash: hashB,
			ExpiresAt: now.Add(30 * 24 * time.Hour),
			CreatedAt: now,
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to persist session B: %v",
			err,
		)
	}

	// Logout using session A.
	body, err := json.Marshal(
		schemas.LogoutRequest{
			RefreshToken: refreshTokenA,
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to encode logout request: %v",
			err,
		)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/logout",
		bytes.NewReader(body),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	request.Header.Set(
		"Authorization",
		"Bearer "+accessToken,
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusNoContent,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	// Session A must now be revoked.
	recordA, err := refreshTokenRepository.FindByTokenHash(
		ctx,
		hashA,
	)
	if err != nil {
		t.Fatalf(
			"failed to retrieve session A: %v",
			err,
		)
	}

	if recordA == nil {
		t.Fatal("expected session A to remain persisted")
	}

	if recordA.RevokedAt == nil {
		t.Fatal("expected session A to be revoked")
	}

	// Session B must remain active.
	recordB, err := refreshTokenRepository.FindByTokenHash(
		ctx,
		hashB,
	)
	if err != nil {
		t.Fatalf(
			"failed to retrieve session B: %v",
			err,
		)
	}

	if recordB == nil {
		t.Fatal("expected session B to remain persisted")
	}

	if recordB.RevokedAt != nil {
		t.Fatal(
			"expected session B to remain active",
		)
	}
}

// setupLogoutIntegration creates the real logout dependencies used
// by tests that only need to verify authentication/error behavior.
//
// The returned cleanup function closes the PostgreSQL connection.
func setupLogoutIntegration(
	t *testing.T,
) (
	http.Handler,
	ports.RefreshTokenRepository,
	ports.TokenService,
	func(),
) {
	t.Helper()

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

	queries := postgres.NewQueries(pool)

	refreshTokenRepository := postgres.NewRefreshTokenRepository(
		pool,
		queries,
	)

	refreshTokenService := security.NewRefreshTokenService()

	tokenService, err := security.NewJWTTokenServiceFromConfig(
		cfg,
	)
	if err != nil {
		pool.Close()

		t.Fatalf(
			"failed to create JWT token service: %v",
			err,
		)
	}

	logoutUserUseCase := use_cases.NewLogoutUserUseCase(
		refreshTokenRepository,
		refreshTokenService,
	)

	logoutUserHandler := handlers.NewLogoutUserHandler(
		logoutUserUseCase,
	)

	router := newLogoutIntegrationRouter(
		logoutUserHandler,
		tokenService,
	)

	return router, refreshTokenRepository, tokenService, pool.Close
}
