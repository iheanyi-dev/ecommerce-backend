package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/iheanyi-dev/ecommerce-backend/gateway/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/gateway/presentation/middleware"
)

// mockIdentityTokenValidator lets us test the propagation middleware
// without depending on the JWT implementation.
//
// The JWT validator itself is already covered by its dedicated tests.
type mockIdentityTokenValidator struct {
	identity ports.AuthenticatedIdentity
	err      error

	receivedToken string
}

func (m *mockIdentityTokenValidator) ValidateAccessToken(
	_ context.Context,
	token string,
) (ports.AuthenticatedIdentity, error) {
	m.receivedToken = token

	return m.identity, m.err
}

func TestIdentityPropagationMiddleware(t *testing.T) {
	t.Run("forwards anonymous request without identity headers", func(t *testing.T) {
		validator := &mockIdentityTokenValidator{}

		authMiddleware := middleware.NewIdentityPropagationMiddleware(validator)

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Empty(
				t,
				r.Header.Get(middleware.AuthenticatedUserIDHeader),
			)
			require.Empty(
				t,
				r.Header.Get(middleware.AuthenticatedRoleHeader),
			)

			w.WriteHeader(http.StatusNoContent)
		})

		handler := authMiddleware.Middleware(next)

		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/stores",
			nil,
		)

		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNoContent, rec.Code)
		require.Empty(t, validator.receivedToken)
	})

	t.Run("strips client supplied identity headers from anonymous request", func(t *testing.T) {
		validator := &mockIdentityTokenValidator{}

		authMiddleware := middleware.NewIdentityPropagationMiddleware(validator)

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Empty(
				t,
				r.Header.Get(middleware.AuthenticatedUserIDHeader),
			)
			require.Empty(
				t,
				r.Header.Get(middleware.AuthenticatedRoleHeader),
			)

			w.WriteHeader(http.StatusNoContent)
		})

		handler := authMiddleware.Middleware(next)

		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/stores",
			nil,
		)

		// These values come from the client and must never be trusted.
		req.Header.Set(
			middleware.AuthenticatedUserIDHeader,
			"attacker-user-id",
		)
		req.Header.Set(
			middleware.AuthenticatedRoleHeader,
			"admin",
		)

		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("valid JWT adds trusted identity headers", func(t *testing.T) {
		validator := &mockIdentityTokenValidator{
			identity: ports.AuthenticatedIdentity{
				UserID: "user-123",
				Role:   "vendor",
			},
		}

		authMiddleware := middleware.NewIdentityPropagationMiddleware(validator)

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(
				t,
				"user-123",
				r.Header.Get(middleware.AuthenticatedUserIDHeader),
			)
			require.Equal(
				t,
				"vendor",
				r.Header.Get(middleware.AuthenticatedRoleHeader),
			)

			w.WriteHeader(http.StatusNoContent)
		})

		handler := authMiddleware.Middleware(next)

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/stores",
			nil,
		)
		req.Header.Set("Authorization", "Bearer valid-token")

		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNoContent, rec.Code)
		require.Equal(t, "valid-token", validator.receivedToken)
	})

	t.Run("valid JWT overwrites client supplied identity headers", func(t *testing.T) {
		validator := &mockIdentityTokenValidator{
			identity: ports.AuthenticatedIdentity{
				UserID: "real-user",
				Role:   "vendor",
			},
		}

		authMiddleware := middleware.NewIdentityPropagationMiddleware(validator)

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(
				t,
				"real-user",
				r.Header.Get(middleware.AuthenticatedUserIDHeader),
			)
			require.Equal(
				t,
				"vendor",
				r.Header.Get(middleware.AuthenticatedRoleHeader),
			)

			w.WriteHeader(http.StatusNoContent)
		})

		handler := authMiddleware.Middleware(next)

		req := httptest.NewRequest(
			http.MethodPatch,
			"/api/v1/stores/store-123",
			nil,
		)

		req.Header.Set("Authorization", "Bearer valid-token")

		// Attempted identity spoofing by the external client.
		req.Header.Set(
			middleware.AuthenticatedUserIDHeader,
			"attacker",
		)
		req.Header.Set(
			middleware.AuthenticatedRoleHeader,
			"admin",
		)

		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("rejects malformed authorization header", func(t *testing.T) {
		validator := &mockIdentityTokenValidator{}

		authMiddleware := middleware.NewIdentityPropagationMiddleware(validator)

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler must not be called")
		})

		handler := authMiddleware.Middleware(next)

		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/stores/me",
			nil,
		)
		req.Header.Set("Authorization", "Basic credentials")

		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Empty(t, validator.receivedToken)
	})

	t.Run("rejects invalid JWT", func(t *testing.T) {
		validator := &mockIdentityTokenValidator{
			err: context.Canceled,
		}

		authMiddleware := middleware.NewIdentityPropagationMiddleware(validator)

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler must not be called")
		})

		handler := authMiddleware.Middleware(next)

		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/stores/me",
			nil,
		)
		req.Header.Set("Authorization", "Bearer invalid-token")

		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Equal(t, "invalid-token", validator.receivedToken)
	})

	t.Run("rejects empty bearer token", func(t *testing.T) {
		validator := &mockIdentityTokenValidator{}

		authMiddleware := middleware.NewIdentityPropagationMiddleware(validator)

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler must not be called")
		})

		handler := authMiddleware.Middleware(next)

		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/stores",
			nil,
		)
		req.Header.Set("Authorization", "Bearer ")

		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Empty(t, validator.receivedToken)
	})
}
