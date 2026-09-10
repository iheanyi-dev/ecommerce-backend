package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// authorizationObservabilityLogger captures structured log events so the
// authorization middleware can be tested without depending on the real
// infrastructure logger.
type authorizationObservabilityLogger struct {
	events []ports.LogEvent
	err    error
}

func (l *authorizationObservabilityLogger) Log(
	ctx context.Context,
	event ports.LogEvent,
) error {
	l.events = append(l.events, event)

	return l.err
}

var _ ports.Logger = (*authorizationObservabilityLogger)(nil)

func TestRequireRoles_LogsAuthorizationDenial(t *testing.T) {
	logger := &authorizationObservabilityLogger{}

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		t.Fatal("expected forbidden request to stop before the next handler")
	})

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/admin/users",
		nil,
	)

	req.Pattern = "/api/v1/admin/users"

	req = req.WithContext(
		middleware.WithAuthenticatedIdentity(
			req.Context(),
			ports.AuthenticatedIdentity{
				UserID: "user-123",
				Role:   "vendor",
			},
		),
	)

	rec := httptest.NewRecorder()

	handler := middleware.RequireRolesWithLogger(
		logger,
		"admin",
	)(next)

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Len(t, logger.events, 1)

	event := logger.events[0]

	assert.Equal(t, "http.authorization.denied", event.Event)
	assert.Equal(t, "/api/v1/admin/users", event.Operation)
	assert.Equal(t, "user-123", event.UserID)
	assert.Equal(t, "vendor", event.Role)
	assert.Equal(t, "GET", event.HTTPMethod)
	assert.Equal(t, "/api/v1/admin/users", event.Route)
	assert.Equal(t, http.StatusForbidden, event.StatusCode)
	assert.Equal(t, "admin", event.RequiredRole)
	assert.Equal(t, "client_error", event.FailureCategory)
}

func TestRequireRoles_AuthorizationLoggerFailureDoesNotAffectResponse(t *testing.T) {
	logger := &authorizationObservabilityLogger{
		err: context.Canceled,
	}

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		t.Fatal("expected forbidden request to stop before the next handler")
	})

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/admin/users",
		nil,
	)

	req.Pattern = "/api/v1/admin/users"

	req = req.WithContext(
		middleware.WithAuthenticatedIdentity(
			req.Context(),
			ports.AuthenticatedIdentity{
				UserID: "user-123",
				Role:   "vendor",
			},
		),
	)

	rec := httptest.NewRecorder()

	handler := middleware.RequireRolesWithLogger(
		logger,
		"admin",
	)(next)

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Len(t, logger.events, 1)
}
