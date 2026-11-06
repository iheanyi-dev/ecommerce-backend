package proxy_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/gateway/infrastructure/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderProxy_ForwardsRequestWithTrustedCredentials(t *testing.T) {
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/orders", r.URL.Path)
		assert.Equal(t, "page=2", r.URL.RawQuery)
		assert.Equal(t, "gateway", r.Header.Get("X-Service-Name"))
		assert.Equal(t, "order-secret", r.Header.Get("X-Service-Secret"))
		assert.Equal(t, "user-123", r.Header.Get("X-Authenticated-User-ID"))
		assert.Equal(t, "customer", r.Header.Get("X-Authenticated-Role"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.JSONEq(t, `{"cart_id":"test"}`, string(body))

		w.WriteHeader(http.StatusCreated)
		_, err = fmt.Fprint(w, `{"status":"created"}`)
		require.NoError(t, err)
	}))
	defer downstream.Close()

	orderProxy, err := proxy.NewOrderProxy(downstream.URL, "gateway", "order-secret")
	require.NoError(t, err)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/orders?page=2",
		strings.NewReader(`{"cart_id":"test"}`),
	)
	request.Header.Set("X-Service-Name", "attacker")
	request.Header.Set("X-Service-Secret", "attacker-secret")
	request.Header.Set("X-Authenticated-User-ID", "user-123")
	request.Header.Set("X-Authenticated-Role", "customer")

	recorder := httptest.NewRecorder()
	orderProxy.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.JSONEq(t, `{"status":"created"}`, recorder.Body.String())
}

func TestOrderProxy_RejectsInvalidURL(t *testing.T) {
	result, err := proxy.NewOrderProxy("://invalid", "gateway", "secret")
	assert.Error(t, err)
	assert.Nil(t, result)
}
