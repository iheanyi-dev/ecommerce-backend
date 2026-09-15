package proxy_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/gateway/infrastructure/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdentityProxy_ForwardsRequestWithServiceCredentials(t *testing.T) {
	downstream := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/api/v1/users/login", r.URL.Path)
			assert.Equal(t, "test=value", r.URL.RawQuery)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

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

			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.Equal(
				t,
				`{"email":"test@example.com"}`,
				string(body),
			)

			w.WriteHeader(http.StatusCreated)

			_, err = fmt.Fprint(w, `{"status":"forwarded"}`)
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
	require.NotNil(t, identityProxy)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/login?test=value",
		strings.NewReader(`{"email":"test@example.com"}`),
	)

	request.Header.Set("Content-Type", "application/json")

	// Simulate an external caller attempting to spoof the trusted
	// service-authentication headers. The Gateway must overwrite them.
	request.Header.Set("X-Service-Name", "attacker")
	request.Header.Set("X-Service-Secret", "attacker-secret")

	recorder := httptest.NewRecorder()

	identityProxy.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(
		t,
		`{"status":"forwarded"}`,
		recorder.Body.String(),
	)
}

func TestIdentityProxy_RejectsInvalidURL(t *testing.T) {
	identityProxy, err := proxy.NewIdentityProxy(
		"://invalid",
		"gateway",
		"test-secret",
	)

	assert.Error(t, err)
	assert.Nil(t, identityProxy)
}
