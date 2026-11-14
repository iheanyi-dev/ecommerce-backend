package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/ports"
	promotionmiddleware "github.com/iheanyi-dev/ecommerce-backend/services/promotion/presentation/middleware"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/shared/config"
)

func TestRequestIDMiddleware(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	tests := []struct {
		name       string
		requestID  string
		wantStatus int
		wantID     bool
	}{
		{name: "generates missing request ID", wantStatus: http.StatusNoContent, wantID: true},
		{name: "preserves valid request ID", requestID: uuid.NewString(), wantStatus: http.StatusNoContent, wantID: true},
		{name: "replaces invalid request ID", requestID: "invalid", wantStatus: http.StatusNoContent, wantID: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.requestID != "" {
				req.Header.Set(promotionmiddleware.RequestIDHeader, tt.requestID)
			}
			rec := httptest.NewRecorder()

			promotionmiddleware.NewRequestIDMiddleware().Middleware(next).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			gotID := rec.Header().Get(promotionmiddleware.RequestIDHeader)
			if tt.wantID && uuid.Validate(gotID) != nil {
				t.Fatalf("expected valid request ID, got %q", gotID)
			}
			if tt.requestID != "" && uuid.Validate(tt.requestID) == nil && gotID != tt.requestID {
				t.Fatalf("request ID = %q, want %q", gotID, tt.requestID)
			}
		})
	}
}

func TestIdentityPropagationMiddleware(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := ports.AuthenticatedIdentityFromContext(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if identity.Role != "customer" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	tests := []struct {
		name       string
		userID     string
		role       string
		wantStatus int
	}{
		{name: "valid identity", userID: uuid.NewString(), role: "customer", wantStatus: http.StatusNoContent},
		{name: "anonymous request", wantStatus: http.StatusUnauthorized},
		{name: "missing role", userID: uuid.NewString(), wantStatus: http.StatusUnauthorized},
		{name: "missing user ID", role: "customer", wantStatus: http.StatusUnauthorized},
		{name: "invalid user ID", userID: "invalid", role: "customer", wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.userID != "" {
				req.Header.Set(promotionmiddleware.AuthenticatedUserIDHeader, tt.userID)
			}
			if tt.role != "" {
				req.Header.Set(promotionmiddleware.AuthenticatedRoleHeader, tt.role)
			}
			rec := httptest.NewRecorder()

			promotionmiddleware.NewIdentityPropagationMiddleware().Middleware(next).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestRequireAuthentication(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := promotionmiddleware.RequireAuthentication(next)

	t.Run("rejects anonymous request", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("allows authenticated request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(ports.WithAuthenticatedIdentity(req.Context(), ports.AuthenticatedIdentity{
			UserID: uuid.New(),
			Role:   "customer",
		}))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
		}
	})
}

func TestServiceAuthenticationMiddleware(t *testing.T) {
	cfg := &config.Config{
		ServiceAuthName:   "promotion-client",
		ServiceAuthSecret: "test-secret",
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := promotionmiddleware.NewServiceAuthenticationMiddleware(cfg).RequireServiceAuthentication(next)

	tests := []struct {
		name       string
		service    string
		secret     string
		wantStatus int
	}{
		{name: "valid credentials", service: "promotion-client", secret: "test-secret", wantStatus: http.StatusNoContent},
		{name: "wrong service", service: "other-service", secret: "test-secret", wantStatus: http.StatusUnauthorized},
		{name: "wrong secret", service: "promotion-client", secret: "wrong", wantStatus: http.StatusUnauthorized},
		{name: "missing credentials", wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set(promotionmiddleware.ServiceNameHeader, tt.service)
			req.Header.Set(promotionmiddleware.ServiceSecretHeader, tt.secret)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
