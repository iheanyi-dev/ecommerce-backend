package http_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/gateway/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/gateway/infrastructure/proxy"
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
		requestIDMiddleware,
		observabilityMiddleware,
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
		requestIDMiddleware,
		observabilityMiddleware,
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
		requestIDMiddleware,
		observabilityMiddleware,
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
