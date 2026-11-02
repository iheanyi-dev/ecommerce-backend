package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/presentation/middleware"
)

func TestRequireAuthenticationRejectsMissingIdentity(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	handler := middleware.RequireAuthentication(next)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(recorder, request)

	if called {
		t.Fatal("expected next handler not to be called")
	}

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestRequireAuthenticationPassesAuthenticatedRequest(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	handler := middleware.RequireAuthentication(next)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	identity := ports.AuthenticatedIdentity{
		UserID: uuid.New(),
		Role:   "customer",
	}
	ctx := ports.WithAuthenticatedIdentity(request.Context(), identity)

	handler.ServeHTTP(recorder, request.WithContext(ctx))

	if !called {
		t.Fatal("expected next handler to be called")
	}

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
}
