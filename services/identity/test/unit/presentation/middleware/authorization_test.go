package middleware_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireRoles(t *testing.T) {
	tests := []struct {
		name           string
		identity       *ports.AuthenticatedIdentity
		allowedRoles   []string
		expectedStatus int
		expectedError  string
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
			expectedError:  "forbidden",
		},
		{
			name:           "rejects unauthenticated request",
			identity:       nil,
			allowedRoles:   []string{"admin"},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "authentication required",
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

			// Successful requests do not have an authorization error body.
			if tt.expectedStatus == http.StatusOK {
				return
			}

			// Authorization failures must use the centralized JSON error
			// response format rather than http.Error's plain-text format.
			assert.Equal(
				t,
				"application/json",
				rec.Header().Get("Content-Type"),
			)

			var response struct {
				Error string `json:"error"`
			}

			require.NoError(
				t,
				json.Unmarshal(rec.Body.Bytes(), &response),
			)

			assert.Equal(
				t,
				tt.expectedError,
				response.Error,
			)
		})
	}
}
func TestRequireRolesWithObservability_RecordsAuthorizationDenialMetric(
	t *testing.T,
) {
	metrics := &authorizationMetricsRecorder{}

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		t.Fatal("expected forbidden request to stop before the next handler")
	})

	req := httptest.NewRequest(
		http.MethodGet,
		"/protected",
		nil,
	)

	req = req.WithContext(
		middleware.WithAuthenticatedIdentity(
			req.Context(),
			ports.AuthenticatedIdentity{
				UserID: "user-123",
				Role:   "user",
			},
		),
	)

	rec := httptest.NewRecorder()

	handler := middleware.RequireRolesWithObservability(
		nil,
		metrics,
		"admin",
	)(next)

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Len(t, metrics.metrics, 1)

	metric := metrics.metrics[0]

	assert.Equal(t, "auth.authorization", metric.Name)
	assert.Equal(t, float64(1), metric.Value)
	assert.Equal(
		t,
		map[string]string{
			"result": "denied",
		},
		metric.Labels,
	)
}

// authorizationMetricsRecorder captures metrics without depending on the
// infrastructure metrics implementation.
type authorizationMetricsRecorder struct {
	metrics []ports.Metric
}

func (m *authorizationMetricsRecorder) Increment(
	ctx context.Context,
	metric ports.Metric,
) error {
	m.metrics = append(m.metrics, metric)
	return nil
}

func (m *authorizationMetricsRecorder) Observe(
	ctx context.Context,
	metric ports.Metric,
) error {
	m.metrics = append(m.metrics, metric)
	return nil
}

var _ ports.Metrics = (*authorizationMetricsRecorder)(nil)
