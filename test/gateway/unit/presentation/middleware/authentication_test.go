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

// mockAccessTokenValidator allows the middleware tests to focus exclusively
// on HTTP authentication behavior without depending on JWT implementation.
type mockAccessTokenValidator struct {
	identity ports.AuthenticatedIdentity
	err      error
}

func (m *mockAccessTokenValidator) ValidateAccessToken(
	_ context.Context,
	_ string,
) (ports.AuthenticatedIdentity, error) {
	return m.identity, m.err
}

func TestAuthenticationMiddleware_RequireAuthentication(t *testing.T) {
	t.Run("accepts valid bearer token and stores authenticated identity", func(t *testing.T) {
		validator := &mockAccessTokenValidator{
			identity: ports.AuthenticatedIdentity{
				UserID: "user-123",
				Role:   "user",
			},
		}

		authMiddleware := middleware.NewAuthenticationMiddleware(validator)

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := middleware.AuthenticatedIdentity(r.Context())

			require.True(t, ok)
			require.Equal(t, "user-123", identity.UserID)
			require.Equal(t, "user", identity.Role)

			w.WriteHeader(http.StatusNoContent)
		})

		handler := authMiddleware.RequireAuthentication(next)

		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/stores/me",
			nil,
		)
		req.Header.Set("Authorization", "Bearer valid-token")

		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("rejects missing authorization header", func(t *testing.T) {
		validator := &mockAccessTokenValidator{}

		authMiddleware := middleware.NewAuthenticationMiddleware(validator)

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler must not be called")
		})

		handler := authMiddleware.RequireAuthentication(next)

		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/stores/me",
			nil,
		)

		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("rejects malformed authorization header", func(t *testing.T) {
		validator := &mockAccessTokenValidator{}

		authMiddleware := middleware.NewAuthenticationMiddleware(validator)

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler must not be called")
		})

		handler := authMiddleware.RequireAuthentication(next)

		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/stores/me",
			nil,
		)
		req.Header.Set("Authorization", "Basic credentials")

		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("rejects empty bearer token", func(t *testing.T) {
		validator := &mockAccessTokenValidator{}

		authMiddleware := middleware.NewAuthenticationMiddleware(validator)

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler must not be called")
		})

		handler := authMiddleware.RequireAuthentication(next)

		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/stores/me",
			nil,
		)
		req.Header.Set("Authorization", "Bearer ")

		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("rejects invalid token", func(t *testing.T) {
		validator := &mockAccessTokenValidator{
			err: context.Canceled,
		}

		authMiddleware := middleware.NewAuthenticationMiddleware(validator)

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler must not be called")
		})

		handler := authMiddleware.RequireAuthentication(next)

		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/stores/me",
			nil,
		)
		req.Header.Set("Authorization", "Bearer invalid-token")

		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestAuthenticatedIdentity_ContextRoundTrip(t *testing.T) {
	identity := ports.AuthenticatedIdentity{
		UserID: "user-456",
		Role:   "vendor",
	}

	ctx := middleware.WithAuthenticatedIdentity(
		context.Background(),
		identity,
	)

	result, ok := middleware.AuthenticatedIdentity(ctx)

	require.True(t, ok)
	require.Equal(t, identity, result)
}

func TestAuthenticatedIdentity_MissingFromContext(t *testing.T) {
	identity, ok := middleware.AuthenticatedIdentity(context.Background())

	require.False(t, ok)
	require.Empty(t, identity.UserID)
	require.Empty(t, identity.Role)
}
