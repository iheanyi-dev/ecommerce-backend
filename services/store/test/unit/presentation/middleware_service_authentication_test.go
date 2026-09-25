package presentation_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/middleware"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/shared/config"
	"github.com/stretchr/testify/require"
)

func TestServiceAuthenticationMiddleware_AllowsValidServiceCredentials(t *testing.T) {
	cfg := &config.Config{
		ServiceAuthName:   "billing",
		ServiceAuthSecret: "billing-test-secret",
	}

	middlewareUnderTest := middleware.NewServiceAuthenticationMiddleware(cfg)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	handler := middlewareUnderTest.RequireServiceAuthentication(next)

	request := httptest.NewRequest(http.MethodPatch, "/api/v1/stores/store-id/status", nil)
	request.Header.Set("X-Service-Name", "billing")
	request.Header.Set("X-Service-Secret", "billing-test-secret")

	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusNoContent, response.Code)
}

func TestServiceAuthenticationMiddleware_RejectsMissingCredentials(t *testing.T) {
	cfg := &config.Config{
		ServiceAuthName:   "billing",
		ServiceAuthSecret: "billing-test-secret",
	}

	middlewareUnderTest := middleware.NewServiceAuthenticationMiddleware(cfg)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler must not be called")
	})

	handler := middlewareUnderTest.RequireServiceAuthentication(next)

	request := httptest.NewRequest(http.MethodPatch, "/api/v1/stores/store-id/status", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestServiceAuthenticationMiddleware_RejectsInvalidServiceName(t *testing.T) {
	cfg := &config.Config{
		ServiceAuthName:   "billing",
		ServiceAuthSecret: "billing-test-secret",
	}

	middlewareUnderTest := middleware.NewServiceAuthenticationMiddleware(cfg)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler must not be called")
	})

	handler := middlewareUnderTest.RequireServiceAuthentication(next)

	request := httptest.NewRequest(http.MethodPatch, "/api/v1/stores/store-id/status", nil)
	request.Header.Set("X-Service-Name", "identity")
	request.Header.Set("X-Service-Secret", "billing-test-secret")

	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestServiceAuthenticationMiddleware_RejectsInvalidServiceSecret(t *testing.T) {
	cfg := &config.Config{
		ServiceAuthName:   "billing",
		ServiceAuthSecret: "billing-test-secret",
	}

	middlewareUnderTest := middleware.NewServiceAuthenticationMiddleware(cfg)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler must not be called")
	})

	handler := middlewareUnderTest.RequireServiceAuthentication(next)

	request := httptest.NewRequest(http.MethodPatch, "/api/v1/stores/store-id/status", nil)
	request.Header.Set("X-Service-Name", "billing")
	request.Header.Set("X-Service-Secret", "wrong-secret")

	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusUnauthorized, response.Code)
}
