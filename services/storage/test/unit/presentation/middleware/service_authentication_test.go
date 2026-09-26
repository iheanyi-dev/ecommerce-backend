package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/presentation/middleware"
)

func TestServiceAuthentication(t *testing.T) {
	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	handler := middleware.ServiceAuthentication(
		"gateway",
		"super-secret",
	)(next)

	t.Run("accepts valid credentials", func(t *testing.T) {
		nextCalled = false

		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/storage/public/test.png",
			nil,
		)
		req.Header.Set(middleware.ServiceNameHeader, "gateway")
		req.Header.Set(middleware.ServiceSecretHeader, "super-secret")

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusNoContent {
			t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
		}

		if !nextCalled {
			t.Fatal("expected next handler to be called")
		}
	})

	t.Run("rejects missing credentials", func(t *testing.T) {
		nextCalled = false

		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/storage/public/test.png",
			nil,
		)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
		}

		if nextCalled {
			t.Fatal("expected next handler not to be called")
		}
	})

	t.Run("rejects invalid credentials", func(t *testing.T) {
		nextCalled = false

		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/storage/public/test.png",
			nil,
		)
		req.Header.Set(middleware.ServiceNameHeader, "wrong-service")
		req.Header.Set(middleware.ServiceSecretHeader, "wrong-secret")

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
		}

		if nextCalled {
			t.Fatal("expected next handler not to be called")
		}
	})
}
