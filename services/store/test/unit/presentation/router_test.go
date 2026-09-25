package presentation_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/middleware"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/shared/config"
	"github.com/stretchr/testify/require"
)

const (
	testServiceName   = "test-service"
	testServiceSecret = "test-secret"
	testUserID        = "11111111-1111-1111-1111-111111111111"
)

func stubHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
}

func newRouterTestHandler() http.Handler {
	handler := stubHandler()

	serviceAuthMiddleware := middleware.NewServiceAuthenticationMiddleware(&config.Config{
		ServiceAuthName:   testServiceName,
		ServiceAuthSecret: testServiceSecret,
	})

	return presentation.NewRouter(
		handler,
		handler,
		handler,
		handler,
		handler,
		handler,
		handler,
		handler,
		handler,
		handler,
		handler,
		handler,
		serviceAuthMiddleware,
		nil,
		nil,
		nil,
	)
}

func authenticatedRequest(
	method string,
	path string,
	userAuth bool,
) *http.Request {
	request := httptest.NewRequest(method, path, nil)

	request.Header.Set(
		middleware.ServiceNameHeader,
		testServiceName,
	)
	request.Header.Set(
		middleware.ServiceSecretHeader,
		testServiceSecret,
	)

	if userAuth {
		request.Header.Set(
			middleware.AuthenticatedUserIDHeader,
			testUserID,
		)
		request.Header.Set(
			middleware.AuthenticatedRoleHeader,
			"user",
		)
	}

	return request
}

func TestNewRouter_RegistersRoutes(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		userAuth bool
	}{
		{
			name:     "create store",
			method:   http.MethodPost,
			path:     "/api/v1/stores",
			userAuth: true,
		},
		{
			name:     "list stores",
			method:   http.MethodGet,
			path:     "/api/v1/stores",
			userAuth: false,
		},
		{
			name:     "get store by id",
			method:   http.MethodGet,
			path:     "/api/v1/stores/11111111-1111-1111-1111-111111111111",
			userAuth: false,
		},
		{
			name:     "get store by owner",
			method:   http.MethodGet,
			path:     "/api/v1/stores/owner",
			userAuth: true,
		},
		{
			name:     "get store by slug",
			method:   http.MethodGet,
			path:     "/api/v1/stores/slug/my-store",
			userAuth: false,
		},
		{
			name:     "update store",
			method:   http.MethodPatch,
			path:     "/api/v1/stores/11111111-1111-1111-1111-111111111111",
			userAuth: true,
		},
		{
			name:     "change store image",
			method:   http.MethodPatch,
			path:     "/api/v1/stores/11111111-1111-1111-1111-111111111111/image",
			userAuth: true,
		},
		{
			name:     "change store plan",
			method:   http.MethodPatch,
			path:     "/api/v1/stores/11111111-1111-1111-1111-111111111111/plan",
			userAuth: true,
		},
		{
			name:     "change store status",
			method:   http.MethodPatch,
			path:     "/api/v1/stores/11111111-1111-1111-1111-111111111111/status",
			userAuth: false,
		},
		{
			name:     "add store knowledge",
			method:   http.MethodPost,
			path:     "/api/v1/stores/11111111-1111-1111-1111-111111111111/knowledge",
			userAuth: true,
		},
		{
			name:     "chat",
			method:   http.MethodPost,
			path:     "/api/v1/stores/11111111-1111-1111-1111-111111111111/chat",
			userAuth: true,
		},
		{
			name:     "remove stale knowledge",
			method:   http.MethodDelete,
			path:     "/api/v1/stores/11111111-1111-1111-1111-111111111111/knowledge/stale",
			userAuth: true,
		},
	}

	router := newRouterTestHandler()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := authenticatedRequest(
				tt.method,
				tt.path,
				tt.userAuth,
			)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusNoContent, recorder.Code)
		})
	}
}

func TestNewRouter_RejectsMissingServiceCredentials(t *testing.T) {
	router := newRouterTestHandler()

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/stores",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestNewRouter_RejectsInvalidServiceCredentials(t *testing.T) {
	router := newRouterTestHandler()

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/stores",
		nil,
	)

	request.Header.Set(
		middleware.ServiceNameHeader,
		"wrong-service",
	)
	request.Header.Set(
		middleware.ServiceSecretHeader,
		"wrong-secret",
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestNewRouter_AllowsPublicRouteWithValidServiceCredentials(t *testing.T) {
	router := newRouterTestHandler()

	request := authenticatedRequest(
		http.MethodGet,
		"/api/v1/stores",
		false,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestNewRouter_RejectsProtectedRouteWithoutUserIdentity(t *testing.T) {
	router := newRouterTestHandler()

	request := authenticatedRequest(
		http.MethodPost,
		"/api/v1/stores",
		false,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestNewRouter_AllowsProtectedRouteWithPropagatedUserIdentity(t *testing.T) {
	router := newRouterTestHandler()

	request := authenticatedRequest(
		http.MethodPost,
		"/api/v1/stores",
		true,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestNewRouter_StatusRouteRequiresServiceAuthenticationOnly(t *testing.T) {
	router := newRouterTestHandler()

	request := authenticatedRequest(
		http.MethodPatch,
		"/api/v1/stores/11111111-1111-1111-1111-111111111111/status",
		false,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestNewRouter_RejectsIncompleteUserIdentity(t *testing.T) {
	router := newRouterTestHandler()

	request := authenticatedRequest(
		http.MethodPost,
		"/api/v1/stores",
		false,
	)

	request.Header.Set(
		middleware.AuthenticatedUserIDHeader,
		testUserID,
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestNewRouter_RejectsInvalidUserIdentity(t *testing.T) {
	router := newRouterTestHandler()

	request := authenticatedRequest(
		http.MethodPost,
		"/api/v1/stores",
		false,
	)

	request.Header.Set(
		middleware.AuthenticatedUserIDHeader,
		"not-a-uuid",
	)
	request.Header.Set(
		middleware.AuthenticatedRoleHeader,
		"user",
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestNewRouter_RejectsUnsupportedMethod(t *testing.T) {
	router := newRouterTestHandler()

	request := authenticatedRequest(
		http.MethodDelete,
		"/api/v1/stores/11111111-1111-1111-1111-111111111111",
		false,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusMethodNotAllowed, recorder.Code)
}

func TestNewRouter_RejectsUnknownRoute(t *testing.T) {
	router := newRouterTestHandler()

	request := authenticatedRequest(
		http.MethodGet,
		"/api/v1/stores/unknown/path",
		false,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNotFound, recorder.Code)
}
