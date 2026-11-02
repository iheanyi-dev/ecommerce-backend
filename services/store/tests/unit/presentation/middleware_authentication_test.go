package presentation_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/middleware"
	"github.com/stretchr/testify/require"
)

func TestRequireAuthentication_AllowsAuthenticatedIdentity(t *testing.T) {
	userID := uuid.New()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := ports.AuthenticatedIdentityFromContext(r.Context())
		require.True(t, ok)
		require.Equal(t, userID, identity.UserID)
		require.Equal(t, "user", identity.Role)

		w.WriteHeader(http.StatusNoContent)
	})

	handler := middleware.RequireAuthentication(next)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/stores", nil)
	request = request.WithContext(
		ports.WithAuthenticatedIdentity(
			request.Context(),
			ports.AuthenticatedIdentity{
				UserID: userID,
				Role:   "user",
			},
		),
	)

	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusNoContent, response.Code)
}

func TestRequireAuthentication_RejectsAnonymousRequest(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler must not be called")
	})

	handler := middleware.RequireAuthentication(next)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/stores", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestRequireAuthentication_PreservesRequestContext(t *testing.T) {
	userID := uuid.New()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := ports.AuthenticatedIdentityFromContext(r.Context())
		require.True(t, ok)
		require.Equal(t, userID, identity.UserID)

		w.WriteHeader(http.StatusNoContent)
	})

	handler := middleware.RequireAuthentication(next)

	request := httptest.NewRequest(http.MethodPatch, "/api/v1/stores/id", nil)
	request = request.WithContext(
		ports.WithAuthenticatedIdentity(
			request.Context(),
			ports.AuthenticatedIdentity{UserID: userID, Role: "vendor"},
		),
	)

	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusNoContent, response.Code)
}
