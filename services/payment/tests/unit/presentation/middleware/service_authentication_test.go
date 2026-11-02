package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/presentation/middleware"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/shared/config"
)

func TestServiceAuthenticationMiddleware_AllowsValidServiceCredentials(t *testing.T) {
	cfg := &config.Config{
		ServiceAuthName:   "payment",
		ServiceAuthSecret: "payment-test-secret",
	}

	middlewareUnderTest := middleware.NewServiceAuthenticationMiddleware(cfg)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	handler := middlewareUnderTest.RequireServiceAuthentication(next)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/payments", nil)
	request.Header.Set(middleware.ServiceNameHeader, "payment")
	request.Header.Set(middleware.ServiceSecretHeader, "payment-test-secret")

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusNoContent, response.Code)
}

func TestServiceAuthenticationMiddleware_RejectsInvalidCredentials(t *testing.T) {
	tests := []struct {
		name          string
		serviceName   string
		serviceSecret string
	}{
		{
			name: "missing credentials",
		},
		{
			name:          "invalid service name",
			serviceName:   "identity",
			serviceSecret: "payment-test-secret",
		},
		{
			name:          "invalid service secret",
			serviceName:   "payment",
			serviceSecret: "wrong-secret",
		},
		{
			name:        "empty service secret",
			serviceName: "payment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				ServiceAuthName:   "payment",
				ServiceAuthSecret: "payment-test-secret",
			}

			middlewareUnderTest := middleware.NewServiceAuthenticationMiddleware(cfg)

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("next handler must not be called")
			})

			handler := middlewareUnderTest.RequireServiceAuthentication(next)

			request := httptest.NewRequest(http.MethodPost, "/api/v1/payments", nil)
			if tt.serviceName != "" {
				request.Header.Set(middleware.ServiceNameHeader, tt.serviceName)
			}
			if tt.serviceSecret != "" {
				request.Header.Set(middleware.ServiceSecretHeader, tt.serviceSecret)
			}

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			require.Equal(t, http.StatusUnauthorized, response.Code)
		})
	}
}
