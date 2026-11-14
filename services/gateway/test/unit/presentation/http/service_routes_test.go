package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/gateway/infrastructure/proxy"
	"github.com/iheanyi-dev/ecommerce-backend/services/gateway/presentation/handlers"
	gatewayhttp "github.com/iheanyi-dev/ecommerce-backend/services/gateway/presentation/http"
	"github.com/iheanyi-dev/ecommerce-backend/services/gateway/presentation/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func serviceRouter(order *proxy.OrderProxy, payment *proxy.PaymentProxy) http.Handler {
	return gatewayhttp.NewRouter(
		handlers.NewHealthHandler(),
		nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil,
		order,
		payment,
	)
}

func TestNewRouter_OrderRoute(t *testing.T) {
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/orders/order-id/cancel", r.URL.Path)
		assert.Equal(t, "order", r.Header.Get("X-Service-Name"))
		assert.Equal(t, "trusted-secret", r.Header.Get("X-Service-Secret"))
		assert.Equal(t, "customer-id", r.Header.Get(middleware.AuthenticatedUserIDHeader))
		assert.Equal(t, "customer", r.Header.Get(middleware.AuthenticatedRoleHeader))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer downstream.Close()

	orderProxy, err := proxy.NewOrderProxy(downstream.URL, "order", "trusted-secret")
	require.NoError(t, err)

	router := serviceRouter(orderProxy, nil)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/orders/order-id/cancel", nil)
	request.Header.Set("X-Service-Name", "untrusted")
	request.Header.Set("X-Service-Secret", "untrusted")
	request.Header.Set(middleware.AuthenticatedUserIDHeader, "customer-id")
	request.Header.Set(middleware.AuthenticatedRoleHeader, "customer")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestNewRouter_PaymentCustomerRoute(t *testing.T) {
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/payments/payment-id", r.URL.Path)
		assert.Equal(t, "customer-id", r.Header.Get(middleware.AuthenticatedUserIDHeader))
		assert.Equal(t, "customer", r.Header.Get(middleware.AuthenticatedRoleHeader))
		assert.Empty(t, r.Header.Get("X-Service-Secret"))
		w.WriteHeader(http.StatusOK)
	}))
	defer downstream.Close()

	paymentProxy, err := proxy.NewPaymentProxy(downstream.URL)
	require.NoError(t, err)

	router := serviceRouter(nil, paymentProxy)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/payments/payment-id", nil)
	request.Header.Set(middleware.AuthenticatedUserIDHeader, "customer-id")
	request.Header.Set(middleware.AuthenticatedRoleHeader, "customer")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestNewRouter_PaymentInternalRoutesNotExposed(t *testing.T) {
	var downstreamCalls int
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downstreamCalls++
		w.WriteHeader(http.StatusOK)
	}))
	defer downstream.Close()

	paymentProxy, err := proxy.NewPaymentProxy(downstream.URL)
	require.NoError(t, err)

	router := serviceRouter(nil, paymentProxy)

	tests := []struct {
		name   string
		method string
		path   string
		status int
	}{
		{"payment creation", http.MethodPost, "/api/v1/payments", http.StatusMethodNotAllowed},
		{"payment verification", http.MethodPost, "/api/v1/payments/payment-id/verify", http.StatusNotFound},
		{"payout listing", http.MethodGet, "/api/v1/payouts", http.StatusNotFound},
		{"payout creation", http.MethodPost, "/api/v1/payouts", http.StatusNotFound},
		{"payout verification", http.MethodPost, "/api/v1/payouts/payout-id/verify", http.StatusNotFound},
		{"Paystack webhook", http.MethodPost, "/api/v1/webhooks/paystack", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.path, nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			assert.Equal(t, tt.status, recorder.Code)
		})
	}

	assert.Zero(t, downstreamCalls, "internal Payment routes must not reach Payment")
}
