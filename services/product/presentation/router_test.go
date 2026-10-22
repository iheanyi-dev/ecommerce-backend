package presentation

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/shared/config"
)

func TestRouterRejectsMissingServiceAuthentication(t *testing.T) {
	router := NewRouter(
		&handlers.ProductHandler{},
		&config.Config{
			ServiceAuthName:   "product",
			ServiceAuthSecret: "test-secret",
		},
	)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func TestRouterRejectsPartialIdentity(t *testing.T) {
	router := NewRouter(
		&handlers.ProductHandler{},
		&config.Config{
			ServiceAuthName:   "product",
			ServiceAuthSecret: "test-secret",
		},
	)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/products/me", nil)
	request.Header.Set("X-Service-Name", "product")
	request.Header.Set("X-Service-Secret", "test-secret")
	request.Header.Set("X-Authenticated-User-ID", "00000000-0000-4000-8000-000000000001")

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d for partial identity, got %d",
			http.StatusUnauthorized, response.Code)
	}
}

func TestRouterReturnsNotFoundForUnknownRoute(t *testing.T) {
	router := NewRouter(
		&handlers.ProductHandler{},
		&config.Config{
			ServiceAuthName:   "product",
			ServiceAuthSecret: "test-secret",
		},
	)

	request := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	request.Header.Set("X-Service-Name", "product")
	request.Header.Set("X-Service-Secret", "test-secret")

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}
}
