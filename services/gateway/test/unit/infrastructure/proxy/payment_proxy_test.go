package proxy_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/gateway/infrastructure/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentProxy_ForwardsAuthenticatedRequest(t *testing.T) {
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/payments/payment-id", r.URL.Path)
		assert.Equal(t, "user-123", r.Header.Get("X-Authenticated-User-ID"))
		assert.Equal(t, "customer", r.Header.Get("X-Authenticated-Role"))

		_, err := fmt.Fprint(w, `{"status":"payment"}`)
		require.NoError(t, err)
	}))
	defer downstream.Close()

	paymentProxy, err := proxy.NewPaymentProxy(downstream.URL)
	require.NoError(t, err)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/payments/payment-id",
		nil,
	)
	request.Header.Set("X-Authenticated-User-ID", "user-123")
	request.Header.Set("X-Authenticated-Role", "customer")

	recorder := httptest.NewRecorder()
	paymentProxy.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"status":"payment"}`, recorder.Body.String())
}

func TestPaymentProxy_RejectsInvalidURL(t *testing.T) {
	result, err := proxy.NewPaymentProxy("://invalid")
	assert.Error(t, err)
	assert.Nil(t, result)
}
