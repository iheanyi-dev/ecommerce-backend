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
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/persistence/postgres"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/security"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/schemas"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/shared/config"
)

// TestUpdateUserStatusIntegration verifies the complete administrative
// account-status update flow using the real HTTP handler, authentication
// middleware, application use case, PostgreSQL repository, and database.
func TestUpdateUserStatusIntegration(t *testing.T) {
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

	tokenService, err := security.NewJWTTokenServiceFromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create JWT token service: %v", err)
	}

	updateUserStatusUseCase := use_cases.NewUpdateUserStatusUseCase(
		userRepository,
	)

	updateUserStatusHandler := handlers.NewUpdateUserStatusHandler(
		updateUserStatusUseCase,
	)

	authenticationMiddleware := middleware.NewAuthenticationMiddleware(
		tokenService,
	)
	listUsersHandler := handlers.NewListUsersHandler(
		&mockRouterListUsersService{},
	)

	getUserHandler := handlers.NewGetUserHandler(
		&mockIntegrationGetUserService{},
	)

	router := presentation.NewRouter(
		&handlers.RegisterUserHandler{},
		&handlers.LoginUserHandler{},
		&handlers.RefreshUserHandler{},
		&handlers.MeHandler{},
		&handlers.UpdateUserProfileHandler{},
		authenticationMiddleware,
		&handlers.LogoutUserHandler{},
		listUsersHandler,
		getUserHandler,
		updateUserStatusHandler,
	)

	email := "status-" + uuid.New().String() + "@example.com"

	passwordHasher := security.NewBcryptPasswordHasher()

	registerUserUseCase := use_cases.NewRegisterUserUseCase(
		userRepository,
		passwordHasher,
	)

	registerResult, err := registerUserUseCase.Execute(
		ctx,
		dto.RegisterUserCommand{
			FullName: "Status Integration User",
			Email:    email,
			Password: "SecurePassword123@",
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
			t.Errorf("failed to clean up test user: %v", err)
		}
	})

	// Activate the newly registered account so the domain permits the
	// ACTIVE -> SUSPENDED transition.
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

	adminUserID := user.NewUserID()

	accessToken, err := tokenService.GenerateAccessToken(
		ctx,
		adminUserID,
		user.RoleAdmin,
	)
	if err != nil {
		t.Fatalf("failed to generate admin access token: %v", err)
	}

	requestBody := schemas.UpdateUserStatusRequest{
		Status: string(user.StatusSuspended),
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("failed to encode request body: %v", err)
	}

	request := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/admin/users/"+registerResult.ID+"/status",
		bytes.NewReader(body),
	)

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+accessToken)

	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d; body: %s",
			http.StatusOK,
			response.Code,
			response.Body.String(),
		)
	}

	var result schemas.UpdateUserStatusResponse

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.UserID != registerResult.ID {
		t.Fatalf(
			"expected user ID %q, got %q",
			registerResult.ID,
			result.UserID,
		)
	}

	if result.Status != string(user.StatusSuspended) {
		t.Fatalf(
			"expected status %q, got %q",
			user.StatusSuspended,
			result.Status,
		)
	}

	var persistedStatus string

	err = pool.QueryRow(
		ctx,
		`SELECT status FROM users WHERE id = $1`,
		registerResult.ID,
	).Scan(&persistedStatus)
	if err != nil {
		t.Fatalf("failed to verify persisted status: %v", err)
	}

	if persistedStatus != string(user.StatusSuspended) {
		t.Fatalf(
			"expected persisted status %q, got %q",
			user.StatusSuspended,
			persistedStatus,
		)
	}
}
