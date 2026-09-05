package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
	"github.com/stretchr/testify/assert"
)

func TestMeHandler_ServeHTTP(t *testing.T) {
	t.Run("returns authenticated identity", func(t *testing.T) {
		handler := handlers.NewMeHandler()

		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/users/me",
			nil,
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

		assert.JSONEq(
			t,
			`{
				"user_id": "550e8400-e29b-41d4-a716-446655440000",
				"role": "user"
			}`,
			recorder.Body.String(),
		)

		assert.Equal(
			t,
			"application/json",
			recorder.Header().Get("Content-Type"),
		)
	})

	t.Run("returns unauthorized when identity is missing", func(t *testing.T) {
		handler := handlers.NewMeHandler()

		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/users/me",
			nil,
		)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(
			t,
			http.StatusUnauthorized,
			recorder.Code,
		)
	})

	t.Run("rejects unsupported method", func(t *testing.T) {
		handler := handlers.NewMeHandler()

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/users/me",
			nil,
		)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(
			t,
			http.StatusMethodNotAllowed,
			recorder.Code,
		)
	})

	t.Run("returns vendor identity", func(t *testing.T) {
		handler := handlers.NewMeHandler()

		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/users/me",
			nil,
		)

		identity := ports.AuthenticatedIdentity{
			UserID: "550e8400-e29b-41d4-a716-446655440000",
			Role:   "vendor",
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

		assert.JSONEq(
			t,
			`{
				"user_id": "550e8400-e29b-41d4-a716-446655440000",
				"role": "vendor"
			}`,
			recorder.Body.String(),
		)
	})

	t.Run("returns admin identity", func(t *testing.T) {
		handler := handlers.NewMeHandler()

		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/users/me",
			nil,
		)

		identity := ports.AuthenticatedIdentity{
			UserID: "550e8400-e29b-41d4-a716-446655440000",
			Role:   "admin",
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

		assert.JSONEq(
			t,
			`{
				"user_id": "550e8400-e29b-41d4-a716-446655440000",
				"role": "admin"
			}`,
			recorder.Body.String(),
		)
	})
}
