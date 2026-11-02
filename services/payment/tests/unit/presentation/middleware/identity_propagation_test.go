package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/presentation/middleware"
)

func TestIdentityPropagationRejectsAnonymousRequest(t *testing.T) {
	called := false
	handler := middleware.NewIdentityPropagationMiddleware().Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}),
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(recorder, request)

	if called {
		t.Fatal("expected anonymous request not to reach next handler")
	}
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestIdentityPropagationAttachesValidIdentity(t *testing.T) {
	userID := uuid.New()
	called := false

	handler := middleware.NewIdentityPropagationMiddleware().Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true

			identity, ok := ports.AuthenticatedIdentityFromContext(r.Context())
			if !ok {
				t.Fatal("expected authenticated identity in context")
			}
			if identity.UserID != userID {
				t.Fatalf("expected user ID %s, got %s", userID, identity.UserID)
			}
			if identity.Role != "customer" {
				t.Fatalf("expected role customer, got %q", identity.Role)
			}

			w.WriteHeader(http.StatusNoContent)
		}),
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(middleware.AuthenticatedUserIDHeader, " "+userID.String()+" ")
	request.Header.Set(middleware.AuthenticatedRoleHeader, " customer ")

	handler.ServeHTTP(recorder, request)

	if !called {
		t.Fatal("expected valid identity request to reach next handler")
	}
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
}

func TestIdentityPropagationRejectsIncompleteIdentity(t *testing.T) {
	tests := []struct {
		name   string
		userID string
		role   string
	}{
		{name: "missing user ID", role: "customer"},
		{name: "missing role", userID: uuid.NewString()},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			called := false
			handler := middleware.NewIdentityPropagationMiddleware().Middleware(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					called = true
				}),
			)

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header.Set(middleware.AuthenticatedUserIDHeader, test.userID)
			request.Header.Set(middleware.AuthenticatedRoleHeader, test.role)

			handler.ServeHTTP(recorder, request)

			if called {
				t.Fatal("expected invalid identity request not to reach next handler")
			}
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
			}
		})
	}
}

func TestIdentityPropagationRejectsMalformedUserID(t *testing.T) {
	called := false
	handler := middleware.NewIdentityPropagationMiddleware().Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}),
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(middleware.AuthenticatedUserIDHeader, "not-a-uuid")
	request.Header.Set(middleware.AuthenticatedRoleHeader, "customer")

	handler.ServeHTTP(recorder, request)

	if called {
		t.Fatal("expected malformed identity request not to reach next handler")
	}
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}
