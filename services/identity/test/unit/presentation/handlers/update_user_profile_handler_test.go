package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeUpdateUserProfileService is a test double for the application port.
//
// The handler test should verify that presentation code communicates with
// the application layer through the port rather than depending on the
// concrete use case.
type fakeUpdateUserProfileService struct {
	called bool

	userID  string
	command dto.UpdateUserProfileCommand

	result dto.UpdateUserProfileResult
	err    error
}

func (f *fakeUpdateUserProfileService) Execute(
	ctx context.Context,
	userID string,
	command dto.UpdateUserProfileCommand,
) (dto.UpdateUserProfileResult, error) {
	f.called = true
	f.userID = userID
	f.command = command

	return f.result, f.err
}

func TestUpdateUserProfileHandler_ServeHTTP(t *testing.T) {
	t.Run("updates the authenticated user's profile", func(t *testing.T) {
		service := &fakeUpdateUserProfileService{
			result: dto.UpdateUserProfileResult{
				UserID:    "550e8400-e29b-41d4-a716-446655440000",
				FullName:  "Updated User",
				Email:     "user@example.com",
				Role:      "user",
				Status:    "active",
				UpdatedAt: "2026-09-06T12:00:00.000Z",
			},
		}

		handler := handlers.NewUpdateUserProfileHandler(service)

		requestBody := `{
			"full_name": "Updated User"
		}`

		req := httptest.NewRequest(
			http.MethodPatch,
			"/api/v1/users/me",
			bytes.NewBufferString(requestBody),
		)

		identity := ports.AuthenticatedIdentity{
			UserID: "550e8400-e29b-41d4-a716-446655440000",
			Role:   "user",
		}

		req = req.WithContext(
			middleware.WithAuthenticatedIdentity(
				req.Context(),
				identity,
			),
		)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(
			t,
			http.StatusOK,
			recorder.Code,
		)

		assert.True(
			t,
			service.called,
		)

		assert.Equal(
			t,
			identity.UserID,
			service.userID,
		)

		assert.Equal(
			t,
			"Updated User",
			service.command.FullName,
		)

		var response map[string]any

		err := json.Unmarshal(
			recorder.Body.Bytes(),
			&response,
		)

		require.NoError(t, err)

		assert.Equal(
			t,
			"550e8400-e29b-41d4-a716-446655440000",
			response["user_id"],
		)

		assert.Equal(
			t,
			"Updated User",
			response["full_name"],
		)

		assert.Equal(
			t,
			"user@example.com",
			response["email"],
		)
	})

	t.Run("rejects request when authenticated identity is missing", func(t *testing.T) {
		service := &fakeUpdateUserProfileService{}

		handler := handlers.NewUpdateUserProfileHandler(service)

		req := httptest.NewRequest(
			http.MethodPatch,
			"/api/v1/users/me",
			bytes.NewBufferString(`{
				"full_name": "Updated User"
			}`),
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
			"update service must not be called without an authenticated identity",
		)
	})

	t.Run("rejects unsupported method", func(t *testing.T) {
		service := &fakeUpdateUserProfileService{}

		handler := handlers.NewUpdateUserProfileHandler(service)

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/users/me",
			bytes.NewBufferString(`{
				"full_name": "Updated User"
			}`),
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

	t.Run("rejects invalid JSON", func(t *testing.T) {
		service := &fakeUpdateUserProfileService{}

		handler := handlers.NewUpdateUserProfileHandler(service)

		req := httptest.NewRequest(
			http.MethodPatch,
			"/api/v1/users/me",
			bytes.NewBufferString(`{"full_name":`),
		)

		identity := ports.AuthenticatedIdentity{
			UserID: "550e8400-e29b-41d4-a716-446655440000",
			Role:   "user",
		}

		req = req.WithContext(
			middleware.WithAuthenticatedIdentity(
				req.Context(),
				identity,
			),
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

	t.Run("does not allow user ID to be supplied by the request", func(t *testing.T) {
		service := &fakeUpdateUserProfileService{
			result: dto.UpdateUserProfileResult{
				UserID:    "550e8400-e29b-41d4-a716-446655440000",
				FullName:  "Updated User",
				Email:     "user@example.com",
				Role:      "user",
				Status:    "active",
				UpdatedAt: "2026-09-06T12:00:00.000Z",
			},
		}

		handler := handlers.NewUpdateUserProfileHandler(service)

		// Deliberately include a different user_id in the request.
		//
		// The handler must ignore/reject this field and use the authenticated
		// identity from request context instead.
		requestBody := `{
			"user_id": "different-user-id",
			"full_name": "Updated User"
		}`

		req := httptest.NewRequest(
			http.MethodPatch,
			"/api/v1/users/me",
			bytes.NewBufferString(requestBody),
		)

		identity := ports.AuthenticatedIdentity{
			UserID: "550e8400-e29b-41d4-a716-446655440000",
			Role:   "user",
		}

		req = req.WithContext(
			middleware.WithAuthenticatedIdentity(
				req.Context(),
				identity,
			),
		)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(
			t,
			http.StatusOK,
			recorder.Code,
		)

		require.True(t, service.called)

		assert.Equal(
			t,
			identity.UserID,
			service.userID,
			"the authenticated identity must determine which user is updated",
		)

		assert.NotEqual(
			t,
			"different-user-id",
			service.userID,
		)
	})

	t.Run("maps user not found to not found", func(t *testing.T) {
		service := &fakeUpdateUserProfileService{
			err: use_cases.ErrUserNotFound,
		}

		handler := handlers.NewUpdateUserProfileHandler(service)

		req := httptest.NewRequest(
			http.MethodPatch,
			"/api/v1/users/me",
			bytes.NewBufferString(`{
				"full_name": "Updated User"
			}`),
		)

		identity := ports.AuthenticatedIdentity{
			UserID: "550e8400-e29b-41d4-a716-446655440000",
			Role:   "user",
		}

		req = req.WithContext(
			middleware.WithAuthenticatedIdentity(
				req.Context(),
				identity,
			),
		)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(
			t,
			http.StatusNotFound,
			recorder.Code,
		)
	})
}
