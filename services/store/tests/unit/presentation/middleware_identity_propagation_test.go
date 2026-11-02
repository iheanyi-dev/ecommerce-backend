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

func TestIdentityPropagationMiddleware_PropagatesTrustedIdentity(t *testing.T) {
	userID := uuid.New()

	var received ports.AuthenticatedIdentity
	var found bool

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received, found = ports.AuthenticatedIdentityFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})

	handler := middleware.NewIdentityPropagationMiddleware().Middleware(next)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/stores/owner", nil)
	request.Header.Set(middleware.AuthenticatedUserIDHeader, userID.String())
	request.Header.Set(middleware.AuthenticatedRoleHeader, "vendor")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.True(t, found)
	require.Equal(t, userID, received.UserID)
	require.Equal(t, "vendor", received.Role)
}

func TestIdentityPropagationMiddleware_AllowsAnonymousRequest(t *testing.T) {
	var found bool

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, found = ports.AuthenticatedIdentityFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})

	handler := middleware.NewIdentityPropagationMiddleware().Middleware(next)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/stores", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.False(t, found)
}

func TestIdentityPropagationMiddleware_RejectsInvalidUserID(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	handler := middleware.NewIdentityPropagationMiddleware().Middleware(next)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/stores/owner", nil)
	request.Header.Set(middleware.AuthenticatedUserIDHeader, "not-a-uuid")
	request.Header.Set(middleware.AuthenticatedRoleHeader, "vendor")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestIdentityPropagationMiddleware_RejectsIncompleteIdentity(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	handler := middleware.NewIdentityPropagationMiddleware().Middleware(next)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/stores/owner", nil)
	request.Header.Set(middleware.AuthenticatedUserIDHeader, uuid.NewString())

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}
