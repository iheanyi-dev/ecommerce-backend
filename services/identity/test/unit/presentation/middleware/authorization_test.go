package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
	"github.com/stretchr/testify/assert"
)

func TestRequireRoles(t *testing.T) {
	tests := []struct {
		name           string
		identity       *ports.AuthenticatedIdentity
		allowedRoles   []string
		expectedStatus int
	}{
		{
			name: "allows admin",
			identity: &ports.AuthenticatedIdentity{
				Role: "admin",
			},
			allowedRoles:   []string{"admin"},
			expectedStatus: http.StatusOK,
		},
		{
			name: "allows vendor",
			identity: &ports.AuthenticatedIdentity{
				Role: "vendor",
			},
			allowedRoles:   []string{"vendor"},
			expectedStatus: http.StatusOK,
		},
		{
			name: "allows user when user role is permitted",
			identity: &ports.AuthenticatedIdentity{
				Role: "user",
			},
			allowedRoles:   []string{"user"},
			expectedStatus: http.StatusOK,
		},
		{
			name: "allows one of multiple permitted roles",
			identity: &ports.AuthenticatedIdentity{
				Role: "vendor",
			},
			allowedRoles:   []string{"admin", "vendor"},
			expectedStatus: http.StatusOK,
		},
		{
			name: "rejects authenticated user with forbidden role",
			identity: &ports.AuthenticatedIdentity{
				Role: "user",
			},
			allowedRoles:   []string{"admin", "vendor"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "rejects unauthenticated request",
			identity:       nil,
			allowedRoles:   []string{"admin"},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false

			next := http.HandlerFunc(func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := middleware.RequireRoles(
				tt.allowedRoles...,
			)(next)

			req := httptest.NewRequest(
				http.MethodGet,
				"/protected",
				nil,
			)

			// Simulate an already authenticated request by placing the
			// identity into the request context.
			if tt.identity != nil {
				ctx := middleware.WithAuthenticatedIdentity(
					req.Context(),
					*tt.identity,
				)

				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(
				t,
				tt.expectedStatus,
				rec.Code,
			)

			// The protected handler should only execute when authorization
			// succeeds.
			assert.Equal(
				t,
				tt.expectedStatus == http.StatusOK,
				nextCalled,
			)
		})
	}
}
