package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/presentation/middleware"
)

func TestRequestID(t *testing.T) {
	t.Run("preserves an existing request ID", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get(middleware.RequestIDHeader); got != "request-123" {
				t.Fatalf("expected request ID request-123, got %q", got)
			}
			w.WriteHeader(http.StatusNoContent)
		})

		handler := middleware.RequestID(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(middleware.RequestIDHeader, "request-123")

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusNoContent {
			t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
		}

		if got := recorder.Header().Get(middleware.RequestIDHeader); got != "request-123" {
			t.Fatalf("expected response request ID request-123, got %q", got)
		}
	})

	t.Run("generates a request ID when absent", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})

		handler := middleware.RequestID(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusNoContent {
			t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
		}

		if recorder.Header().Get(middleware.RequestIDHeader) == "" {
			t.Fatal("expected generated request ID in response")
		}
	})
}
