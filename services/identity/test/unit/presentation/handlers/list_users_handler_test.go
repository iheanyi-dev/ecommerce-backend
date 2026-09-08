package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeListUsersService is a test double for the application service port.
//
// The handler must communicate with the application layer through the
// ListUsersService abstraction rather than depending on the concrete
// ListUsersUseCase.
type fakeListUsersService struct {
	called bool

	gotLimit  int
	gotOffset int

	result *dto.ListUsersResult
	err    error
}

func (f *fakeListUsersService) Execute(
	ctx context.Context,
	limit int,
	offset int,
) (*dto.ListUsersResult, error) {
	f.called = true
	f.gotLimit = limit
	f.gotOffset = offset

	return f.result, f.err
}

func authenticatedRequest(
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

func TestListUsersHandler_ServeHTTP(t *testing.T) {
	t.Run("lists users for an authenticated administrator", func(t *testing.T) {
		service := &fakeListUsersService{
			result: &dto.ListUsersResult{
				Users: []dto.UserSummary{
					{
						ID:       "550e8400-e29b-41d4-a716-446655440000",
						FullName: "John Doe",
						Email:    "john@example.com",
						Role:     "user",
						Status:   "active",
					},
					{
						ID:       "650e8400-e29b-41d4-a716-446655440000",
						FullName: "Jane Doe",
						Email:    "jane@example.com",
						Role:     "vendor",
						Status:   "active",
					},
				},
			},
		}

		handler := handlers.NewListUsersHandler(service)

		identity := ports.AuthenticatedIdentity{
			UserID: "admin-user-id",
			Role:   "admin",
		}

		req := authenticatedRequest(
			http.MethodGet,
			"/api/v1/admin/users?page=1&limit=20",
			identity,
		)

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
			20,
			service.gotLimit,
		)

		assert.Equal(
			t,
			0,
			service.gotOffset,
		)

		var response struct {
			Users []struct {
				ID        string `json:"id"`
				FullName  string `json:"full_name"`
				Email     string `json:"email"`
				Role      string `json:"role"`
				Status    string `json:"status"`
				CreatedAt string `json:"created_at"`
				UpdatedAt string `json:"updated_at"`
			} `json:"users"`
		}

		err := json.Unmarshal(
			recorder.Body.Bytes(),
			&response,
		)

		require.NoError(t, err)

		require.Len(
			t,
			response.Users,
			2,
		)

		assert.Equal(
			t,
			"550e8400-e29b-41d4-a716-446655440000",
			response.Users[0].ID,
		)

		assert.Equal(
			t,
			"John Doe",
			response.Users[0].FullName,
		)

		assert.Equal(
			t,
			"john@example.com",
			response.Users[0].Email,
		)

		assert.Equal(
			t,
			"user",
			response.Users[0].Role,
		)

		assert.Equal(
			t,
			"active",
			response.Users[0].Status,
		)
	})

	t.Run("converts page and limit into repository pagination", func(t *testing.T) {
		service := &fakeListUsersService{
			result: &dto.ListUsersResult{},
		}

		handler := handlers.NewListUsersHandler(service)

		identity := ports.AuthenticatedIdentity{
			UserID: "admin-user-id",
			Role:   "admin",
		}

		req := authenticatedRequest(
			http.MethodGet,
			"/api/v1/admin/users?page=3&limit=25",
			identity,
		)

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
			25,
			service.gotLimit,
		)

		// Page 3 with 25 items per page:
		//
		// offset = (page - 1) * limit
		// offset = (3 - 1) * 25
		// offset = 50
		assert.Equal(
			t,
			50,
			service.gotOffset,
		)
	})

	t.Run("uses default pagination when query parameters are omitted", func(t *testing.T) {
		service := &fakeListUsersService{
			result: &dto.ListUsersResult{},
		}

		handler := handlers.NewListUsersHandler(service)

		identity := ports.AuthenticatedIdentity{
			UserID: "admin-user-id",
			Role:   "admin",
		}

		req := authenticatedRequest(
			http.MethodGet,
			"/api/v1/admin/users",
			identity,
		)

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

		// The handler will use the endpoint's default page size.
		assert.Greater(
			t,
			service.gotLimit,
			0,
		)

		assert.Equal(
			t,
			0,
			service.gotOffset,
		)
	})

	t.Run("rejects request when authenticated identity is missing", func(t *testing.T) {
		service := &fakeListUsersService{}

		handler := handlers.NewListUsersHandler(service)

		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/admin/users",
			nil,
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
			"list users service must not be called without authentication",
		)
	})

	t.Run("rejects unsupported method", func(t *testing.T) {
		service := &fakeListUsersService{}

		handler := handlers.NewListUsersHandler(service)

		identity := ports.AuthenticatedIdentity{
			UserID: "admin-user-id",
			Role:   "admin",
		}

		req := authenticatedRequest(
			http.MethodPost,
			"/api/v1/admin/users",
			identity,
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

	t.Run("rejects invalid page", func(t *testing.T) {
		service := &fakeListUsersService{}

		handler := handlers.NewListUsersHandler(service)

		identity := ports.AuthenticatedIdentity{
			UserID: "admin-user-id",
			Role:   "admin",
		}

		req := authenticatedRequest(
			http.MethodGet,
			"/api/v1/admin/users?page=invalid&limit=20",
			identity,
		)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(
			t,
			http.StatusBadRequest,
			recorder.Code,
		)

		assert.False(
			t,
			service.called,
		)
	})

	t.Run("rejects invalid limit", func(t *testing.T) {
		service := &fakeListUsersService{}

		handler := handlers.NewListUsersHandler(service)

		identity := ports.AuthenticatedIdentity{
			UserID: "admin-user-id",
			Role:   "admin",
		}

		req := authenticatedRequest(
			http.MethodGet,
			"/api/v1/admin/users?page=1&limit=invalid",
			identity,
		)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(
			t,
			http.StatusBadRequest,
			recorder.Code,
		)

		assert.False(
			t,
			service.called,
		)
	})

	t.Run("rejects zero page", func(t *testing.T) {
		service := &fakeListUsersService{}

		handler := handlers.NewListUsersHandler(service)

		identity := ports.AuthenticatedIdentity{
			UserID: "admin-user-id",
			Role:   "admin",
		}

		req := authenticatedRequest(
			http.MethodGet,
			"/api/v1/admin/users?page=0&limit=20",
			identity,
		)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(
			t,
			http.StatusBadRequest,
			recorder.Code,
		)

		assert.False(
			t,
			service.called,
		)
	})

	t.Run("rejects negative limit", func(t *testing.T) {
		service := &fakeListUsersService{}

		handler := handlers.NewListUsersHandler(service)

		identity := ports.AuthenticatedIdentity{
			UserID: "admin-user-id",
			Role:   "admin",
		}

		req := authenticatedRequest(
			http.MethodGet,
			"/api/v1/admin/users?page=1&limit=-10",
			identity,
		)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(
			t,
			http.StatusBadRequest,
			recorder.Code,
		)

		assert.False(
			t,
			service.called,
		)
	})

	t.Run("maps application errors to internal server error", func(t *testing.T) {
		service := &fakeListUsersService{
			err: errors.New("database failure"),
		}

		handler := handlers.NewListUsersHandler(service)

		identity := ports.AuthenticatedIdentity{
			UserID: "admin-user-id",
			Role:   "admin",
		}

		req := authenticatedRequest(
			http.MethodGet,
			"/api/v1/admin/users?page=1&limit=20",
			identity,
		)

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

	t.Run("returns an empty users collection when no users exist", func(t *testing.T) {
		service := &fakeListUsersService{
			result: &dto.ListUsersResult{
				Users: []dto.UserSummary{},
			},
		}

		handler := handlers.NewListUsersHandler(service)

		identity := ports.AuthenticatedIdentity{
			UserID: "admin-user-id",
			Role:   "admin",
		}

		req := authenticatedRequest(
			http.MethodGet,
			"/api/v1/admin/users?page=1&limit=20",
			identity,
		)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(
			t,
			http.StatusOK,
			recorder.Code,
		)

		var response struct {
			Users []any `json:"users"`
		}

		err := json.Unmarshal(
			recorder.Body.Bytes(),
			&response,
		)

		require.NoError(t, err)

		assert.Empty(
			t,
			response.Users,
		)
	})

	t.Run("does not expose password hash", func(t *testing.T) {
		service := &fakeListUsersService{
			result: &dto.ListUsersResult{
				Users: []dto.UserSummary{
					{
						ID:       "550e8400-e29b-41d4-a716-446655440000",
						FullName: "John Doe",
						Email:    "john@example.com",
						Role:     "user",
						Status:   "active",
					},
				},
			},
		}

		handler := handlers.NewListUsersHandler(service)

		identity := ports.AuthenticatedIdentity{
			UserID: "admin-user-id",
			Role:   "admin",
		}

		req := authenticatedRequest(
			http.MethodGet,
			"/api/v1/admin/users?page=1&limit=20",
			identity,
		)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(
			t,
			http.StatusOK,
			recorder.Code,
		)

		// The raw response must not contain the sensitive database field.
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
