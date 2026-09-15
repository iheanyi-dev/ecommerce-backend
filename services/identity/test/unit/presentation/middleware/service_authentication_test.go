package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/shared/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceAuthenticationMiddleware_AllowsValidCredentials(t *testing.T) {
	serviceAuthenticationMiddleware := middleware.NewServiceAuthenticationMiddleware(
		&config.Config{
			ServiceAuthName:   "gateway",
			ServiceAuthSecret: "test-secret",
		},
	)
	nextCalled := false

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	handler := serviceAuthenticationMiddleware.RequireServiceAuthentication(
		next,
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/login",
		nil,
	)

	request.Header.Set("X-Service-Name", "gateway")
	request.Header.Set("X-Service-Secret", "test-secret")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.True(t, nextCalled)
}

func TestServiceAuthenticationMiddleware_RejectsMissingServiceName(t *testing.T) {
	testServiceAuthenticationFailure(
		t,
		"",
		"test-secret",
	)
}

func TestServiceAuthenticationMiddleware_RejectsMissingServiceSecret(t *testing.T) {
	testServiceAuthenticationFailure(
		t,
		"gateway",
		"",
	)
}

func TestServiceAuthenticationMiddleware_RejectsInvalidServiceName(t *testing.T) {
	testServiceAuthenticationFailure(
		t,
		"attacker",
		"test-secret",
	)
}

func TestServiceAuthenticationMiddleware_RejectsInvalidServiceSecret(t *testing.T) {
	testServiceAuthenticationFailure(
		t,
		"gateway",
		"wrong-secret",
	)
}

func testServiceAuthenticationFailure(
	t *testing.T,
	serviceName string,
	serviceSecret string,
) {
	t.Helper()

	serviceAuthenticationMiddleware :=
		middleware.NewServiceAuthenticationMiddleware(
			&config.Config{
				ServiceAuthName:   "gateway",
				ServiceAuthSecret: "test-secret",
			},
		)

	nextCalled := false

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	handler := serviceAuthenticationMiddleware.RequireServiceAuthentication(
		next,
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/login",
		nil,
	)

	request.Header.Set("X-Service-Name", serviceName)
	request.Header.Set("X-Service-Secret", serviceSecret)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.False(t, nextCalled)
	assert.JSONEq(
		t,
		`{"error":"invalid service credentials"}`,
		recorder.Body.String(),
	)
}

func TestServiceAuthenticationMiddleware_DoesNotExposeExpectedCredentials(
	t *testing.T,
) {
	serviceAuthenticationMiddleware :=
		middleware.NewServiceAuthenticationMiddleware(
			&config.Config{
				ServiceAuthName:   "gateway",
				ServiceAuthSecret: "test-secret",
			},
		)

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		t.Fatal("next handler should not be called")
	})

	handler := serviceAuthenticationMiddleware.RequireServiceAuthentication(
		next,
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/login",
		nil,
	)

	request.Header.Set("X-Service-Name", "gateway")
	request.Header.Set("X-Service-Secret", "wrong-secret")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.NotContains(
		t,
		recorder.Body.String(),
		"test-secret",
	)
}
