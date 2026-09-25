package http_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/gateway/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/gateway/infrastructure/proxy"
	"github.com/iheanyi-dev/ecommerce-backend/gateway/infrastructure/security"
	"github.com/iheanyi-dev/ecommerce-backend/gateway/presentation/handlers"
	gatewayhttp "github.com/iheanyi-dev/ecommerce-backend/gateway/presentation/http"
	"github.com/iheanyi-dev/ecommerce-backend/gateway/presentation/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLogger struct{}

func (f *fakeLogger) Info(
	_ context.Context,
	_ string,
	_ ...ports.Field,
) {
}

func (f *fakeLogger) Error(
	_ context.Context,
	_ string,
	_ ...ports.Field,
) {
}

func TestNewRouter_Health(t *testing.T) {
	healthHandler := handlers.NewHealthHandler()

	downstream := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			t.Fatal("Identity service should not receive health requests")
		}),
	)
	defer downstream.Close()

	identityProxy, err := proxy.NewIdentityProxy(
		downstream.URL,
		"gateway",
		"test-secret",
	)
	require.NoError(t, err)

	requestIDMiddleware := middleware.NewRequestIDMiddleware()
	observabilityMiddleware :=
		middleware.NewRequestObservabilityMiddleware(&fakeLogger{})

	router := gatewayhttp.NewRouter(
		healthHandler,
		identityProxy,
		nil,
		requestIDMiddleware,
		observabilityMiddleware,
		nil,
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)

	requestID := recorder.Header().Get("X-Request-ID")
	assert.NotEmpty(t, requestID)
}

func TestNewRouter_IdentityRoute(t *testing.T) {
	downstream := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			assert.Equal(
				t,
				"/api/v1/users/login",
				r.URL.Path,
			)

			assert.NotEmpty(
				t,
				r.Header.Get("X-Request-ID"),
			)

			w.WriteHeader(http.StatusCreated)

			_, err := fmt.Fprint(w, `{"status":"identity"}`)
			require.NoError(t, err)
		}),
	)
	defer downstream.Close()

	identityProxy, err := proxy.NewIdentityProxy(
		downstream.URL,
		"gateway",
		"test-secret",
	)
	require.NoError(t, err)

	healthHandler := handlers.NewHealthHandler()
	requestIDMiddleware := middleware.NewRequestIDMiddleware()
	observabilityMiddleware :=
		middleware.NewRequestObservabilityMiddleware(&fakeLogger{})

	router := gatewayhttp.NewRouter(
		healthHandler,
		identityProxy,
		nil,
		requestIDMiddleware,
		observabilityMiddleware,
		nil,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/login",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, `{"status":"identity"}`, recorder.Body.String())

	requestID := recorder.Header().Get("X-Request-ID")
	assert.NotEmpty(t, requestID)
}

func TestNewRouter_IdentityRoute_BypassesJWTValidation(t *testing.T) {
	downstream := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			// The invalid JWT must reach Identity unchanged.
			assert.Equal(
				t,
				"Bearer invalid-token",
				r.Header.Get("Authorization"),
			)

			// Identity routes bypass Gateway identity propagation,
			// so client-supplied identity headers remain untouched.
			assert.Equal(
				t,
				"client-user-id",
				r.Header.Get(middleware.AuthenticatedUserIDHeader),
			)

			assert.Equal(
				t,
				"client-role",
				r.Header.Get(middleware.AuthenticatedRoleHeader),
			)

			w.WriteHeader(http.StatusOK)

			_, err := fmt.Fprint(
				w,
				`{"status":"identity-bypassed"}`,
			)
			require.NoError(t, err)
		}),
	)
	defer downstream.Close()

	identityProxy, err := proxy.NewIdentityProxy(
		downstream.URL,
		"gateway",
		"test-secret",
	)
	require.NoError(t, err)

	healthHandler := handlers.NewHealthHandler()
	requestIDMiddleware := middleware.NewRequestIDMiddleware()
	observabilityMiddleware :=
		middleware.NewRequestObservabilityMiddleware(&fakeLogger{})

	// Construct the real Gateway JWT validator.
	//
	// The Identity route should bypass this validator completely.
	jwtValidator, err := security.NewJWTValidator(
		"test-secret-that-is-at-least-32-characters-long",
		"identity-service",
	)
	require.NoError(t, err)

	identityPropagationMiddleware :=
		middleware.NewIdentityPropagationMiddleware(jwtValidator)

	router := gatewayhttp.NewRouter(
		healthHandler,
		identityProxy,
		nil,
		requestIDMiddleware,
		observabilityMiddleware,
		identityPropagationMiddleware,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/login",
		nil,
	)

	request.Header.Set(
		"Authorization",
		"Bearer invalid-token",
	)

	request.Header.Set(
		middleware.AuthenticatedUserIDHeader,
		"client-user-id",
	)

	request.Header.Set(
		middleware.AuthenticatedRoleHeader,
		"client-role",
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(
		t,
		`{"status":"identity-bypassed"}`,
		recorder.Body.String(),
	)
}
func TestNewRouter_UnknownRoute(t *testing.T) {
	healthHandler := handlers.NewHealthHandler()

	downstream := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			t.Fatal("Identity service should not receive unknown routes")
		}),
	)
	defer downstream.Close()

	identityProxy, err := proxy.NewIdentityProxy(
		downstream.URL,
		"gateway",
		"test-secret",
	)
	require.NoError(t, err)

	requestIDMiddleware := middleware.NewRequestIDMiddleware()
	observabilityMiddleware :=
		middleware.NewRequestObservabilityMiddleware(&fakeLogger{})

	router := gatewayhttp.NewRouter(
		healthHandler,
		identityProxy,
		nil,
		requestIDMiddleware,
		observabilityMiddleware,
		nil,
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/unknown",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusNotFound, recorder.Code)

	requestID := recorder.Header().Get("X-Request-ID")
	assert.NotEmpty(t, requestID)
}
func TestNewRouter_StoreRoute(t *testing.T) {
	downstream := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			assert.Equal(
				t,
				"/api/v1/stores",
				r.URL.Path,
			)

			assert.Equal(
				t,
				"gateway",
				r.Header.Get("X-Service-Name"),
			)

			assert.Equal(
				t,
				"test-secret",
				r.Header.Get("X-Service-Secret"),
			)

			w.WriteHeader(http.StatusOK)

			_, err := fmt.Fprint(
				w,
				`{"status":"store"}`,
			)
			require.NoError(t, err)
		}),
	)
	defer downstream.Close()

	storeProxy, err := proxy.NewStoreProxy(
		downstream.URL,
		"gateway",
		"test-secret",
	)
	require.NoError(t, err)

	identityProxy, err := proxy.NewIdentityProxy(
		downstream.URL,
		"gateway",
		"test-secret",
	)
	require.NoError(t, err)

	healthHandler := handlers.NewHealthHandler()
	requestIDMiddleware := middleware.NewRequestIDMiddleware()
	observabilityMiddleware :=
		middleware.NewRequestObservabilityMiddleware(&fakeLogger{})

	router := gatewayhttp.NewRouter(
		healthHandler,
		identityProxy,
		storeProxy,
		requestIDMiddleware,
		observabilityMiddleware,
		nil,
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/stores",
		nil,
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(
		t,
		`{"status":"store"}`,
		recorder.Body.String(),
	)
}
