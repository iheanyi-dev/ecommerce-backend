package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeGetUserService is a test double for the application service port.
//
// The handler must communicate with the application layer through the
// GetUserService abstraction rather than depending on the concrete
// GetUserUseCase.
type fakeGetUserService struct {
	called bool

	gotUserID string

	result *dto.GetUserResult
	err    error
}

// Execute records the user ID supplied by the HTTP handler and returns
// the configured test result.
func (f *fakeGetUserService) Execute(
	ctx context.Context,
	userID string,
) (*dto.GetUserResult, error) {
	f.called = true
	f.gotUserID = userID

	return f.result, f.err
}

// authenticatedGetUserRequest creates an HTTP request containing an
// authenticated identity in the request context.
//
// Authentication is normally established by AuthenticationMiddleware.
// The handler test injects the identity directly so the handler can be
// tested independently from the middleware implementation.
func authenticatedGetUserRequest(
	method string,
	target string,
	identity ports.AuthenticatedIdentity,
) *http.Request {
	req := httptest.NewRequest(
		method,
		target,
		nil,
	)

	return req.WithContext(
		middleware.WithAuthenticatedIdentity(
			req.Context(),
			identity,
		),
	)
}

func TestGetUserHandler_ServeHTTP(t *testing.T) {
	t.Run("returns user for an authenticated administrator", func(t *testing.T) {
		createdAt := time.Date(
			2026,
			time.January,
			10,
			12,
			0,
			0,
			0,
			time.UTC,
		)

		updatedAt := time.Date(
			2026,
			time.January,
			11,
			12,
			0,
			0,
			0,
			time.UTC,
		)

		userID := "550e8400-e29b-41d4-a716-446655440000"

		service := &fakeGetUserService{
			result: &dto.GetUserResult{
				ID:        userID,
				FullName:  "John Doe",
				Email:     "john@example.com",
				Role:      "user",
				Status:    "active",
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			},
		}

		handler := handlers.NewGetUserHandler(service)

		identity := ports.AuthenticatedIdentity{
			UserID: "admin-user-id",
			Role:   "admin",
		}

		req := authenticatedGetUserRequest(
			http.MethodGet,
			"/api/v1/admin/users/"+userID,
			identity,
		)

		// Go's HTTP request path value is normally populated by ServeMux
		// route matching. The unit test sets it explicitly because the
		// handler is tested directly without the router.
		req.SetPathValue("id", userID)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(
			t,
			http.StatusOK,
			recorder.Code,
		)

		require.True(
			t,
			service.called,
		)

		assert.Equal(
			t,
			userID,
			service.gotUserID,
		)

		var response struct {
			ID        string    `json:"id"`
			FullName  string    `json:"full_name"`
			Email     string    `json:"email"`
			Role      string    `json:"role"`
			Status    string    `json:"status"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
		}

		err := json.Unmarshal(
			recorder.Body.Bytes(),
			&response,
		)

		require.NoError(t, err)

		assert.Equal(
			t,
			userID,
			response.ID,
		)

		assert.Equal(
			t,
			"John Doe",
			response.FullName,
		)

		assert.Equal(
			t,
			"john@example.com",
			response.Email,
		)

		assert.Equal(
			t,
			"user",
			response.Role,
		)

		assert.Equal(
			t,
			"active",
			response.Status,
		)

		assert.Equal(
			t,
			createdAt,
			response.CreatedAt,
		)

		assert.Equal(
			t,
			updatedAt,
			response.UpdatedAt,
		)
	})

	t.Run("passes path user ID to application service", func(t *testing.T) {
		userID := "650e8400-e29b-41d4-a716-446655440000"

		service := &fakeGetUserService{
			result: &dto.GetUserResult{
				ID:       userID,
				FullName: "Jane Doe",
				Email:    "jane@example.com",
				Role:     "vendor",
				Status:   "active",
			},
		}

		handler := handlers.NewGetUserHandler(service)

		identity := ports.AuthenticatedIdentity{
			UserID: "admin-user-id",
			Role:   "admin",
		}

		req := authenticatedGetUserRequest(
			http.MethodGet,
			"/api/v1/admin/users/"+userID,
			identity,
		)

		req.SetPathValue("id", userID)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(
			t,
			http.StatusOK,
			recorder.Code,
		)

		require.True(
			t,
			service.called,
		)

		assert.Equal(
			t,
			userID,
			service.gotUserID,
		)
	})

	t.Run("rejects request when authenticated identity is missing", func(t *testing.T) {
		service := &fakeGetUserService{}

		handler := handlers.NewGetUserHandler(service)

		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/admin/users/550e8400-e29b-41d4-a716-446655440000",
			nil,
		)

		req.SetPathValue(
			"id",
			"550e8400-e29b-41d4-a716-446655440000",
		)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(
			t,
			http.StatusUnauthorized,
			recorder.Code,
		)

		assert.False(
			t,
			service.called,
			"get user service must not be called without authentication",
		)
	})

	t.Run("rejects unsupported method", func(t *testing.T) {
		service := &fakeGetUserService{}

		handler := handlers.NewGetUserHandler(service)

		identity := ports.AuthenticatedIdentity{
			UserID: "admin-user-id",
			Role:   "admin",
		}

		req := authenticatedGetUserRequest(
			http.MethodPost,
			"/api/v1/admin/users/550e8400-e29b-41d4-a716-446655440000",
			identity,
		)

		req.SetPathValue(
			"id",
			"550e8400-e29b-41d4-a716-446655440000",
		)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(
			t,
			http.StatusMethodNotAllowed,
			recorder.Code,
		)

		assert.False(
			t,
			service.called,
		)
	})

	t.Run("returns not found when application reports missing user", func(t *testing.T) {
		service := &fakeGetUserService{
			err: use_cases.ErrUserNotFound,
		}

		handler := handlers.NewGetUserHandler(service)

		identity := ports.AuthenticatedIdentity{
			UserID: "admin-user-id",
			Role:   "admin",
		}

		userID := "550e8400-e29b-41d4-a716-446655440000"

		req := authenticatedGetUserRequest(
			http.MethodGet,
			"/api/v1/admin/users/"+userID,
			identity,
		)

		req.SetPathValue("id", userID)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(
			t,
			http.StatusNotFound,
			recorder.Code,
		)

		require.True(
			t,
			service.called,
		)
	})

	t.Run("maps unexpected application errors to internal server error", func(t *testing.T) {
		service := &fakeGetUserService{
			err: errors.New("database failure"),
		}

		handler := handlers.NewGetUserHandler(service)

		identity := ports.AuthenticatedIdentity{
			UserID: "admin-user-id",
			Role:   "admin",
		}

		userID := "550e8400-e29b-41d4-a716-446655440000"

		req := authenticatedGetUserRequest(
			http.MethodGet,
			"/api/v1/admin/users/"+userID,
			identity,
		)

		req.SetPathValue("id", userID)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(
			t,
			http.StatusInternalServerError,
			recorder.Code,
		)

		require.True(
			t,
			service.called,
		)
	})

	t.Run("does not expose password hash", func(t *testing.T) {
		userID := "550e8400-e29b-41d4-a716-446655440000"

		service := &fakeGetUserService{
			result: &dto.GetUserResult{
				ID:       userID,
				FullName: "John Doe",
				Email:    "john@example.com",
				Role:     "user",
				Status:   "active",
			},
		}

		handler := handlers.NewGetUserHandler(service)

		identity := ports.AuthenticatedIdentity{
			UserID: "admin-user-id",
			Role:   "admin",
		}

		req := authenticatedGetUserRequest(
			http.MethodGet,
			"/api/v1/admin/users/"+userID,
			identity,
		)

		req.SetPathValue("id", userID)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(
			t,
			http.StatusOK,
			recorder.Code,
		)

		// Password hashes must never cross the presentation boundary.
		assert.NotContains(
			t,
			recorder.Body.String(),
			"password_hash",
		)

		assert.NotContains(
			t,
			recorder.Body.String(),
			"hashed-password",
		)
	})
}
