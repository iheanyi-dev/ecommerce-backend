package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/presentation/middleware"
)

func TestRequestIDPreservesValidID(t *testing.T) {
	requestID := uuid.NewString()
	handler := middleware.NewRequestID().Middleware(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if got := middleware.GetRequestID(r.Context()); got != requestID {
				t.Errorf("expected request ID %s, got %s", requestID, got)
			}
			w.WriteHeader(http.StatusOK)
		},
	))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Request-ID", requestID)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if got := response.Header().Get("X-Request-ID"); got != requestID {
		t.Fatalf("expected response request ID %s, got %s", requestID, got)
	}
}

func TestRequestIDReplacesInvalidID(t *testing.T) {
	handler := middleware.NewRequestID().Middleware(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if _, err := uuid.Parse(middleware.GetRequestID(r.Context())); err != nil {
				t.Errorf("expected generated UUID, got %q", middleware.GetRequestID(r.Context()))
			}
			w.WriteHeader(http.StatusOK)
		},
	))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Request-ID", "invalid")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if _, err := uuid.Parse(response.Header().Get("X-Request-ID")); err != nil {
		t.Fatalf("expected valid response UUID, got %q", response.Header().Get("X-Request-ID"))
	}
}

func TestServiceAuthentication(t *testing.T) {
	handler := middleware.NewServiceAuthentication("order", "secret").Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	)

	tests := []struct {
		name       string
		service    string
		secret     string
		statusCode int
	}{
		{
			name:       "valid credentials",
			service:    "order",
			secret:     "secret",
			statusCode: http.StatusNoContent,
		},
		{
			name:       "invalid service",
			service:    "identity",
			secret:     "secret",
			statusCode: http.StatusUnauthorized,
		},
		{
			name:       "invalid secret",
			service:    "order",
			secret:     "wrong",
			statusCode: http.StatusUnauthorized,
		},
		{
			name:       "missing credentials",
			statusCode: http.StatusUnauthorized,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header.Set("X-Service-Name", test.service)
			request.Header.Set("X-Service-Secret", test.secret)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != test.statusCode {
				t.Fatalf("expected status %d, got %d", test.statusCode, response.Code)
			}
		})
	}
}
