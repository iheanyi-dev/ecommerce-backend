package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
)

type fakeObservabilityLogger struct {
	events []ports.LogEvent
	err    error
}

func (f *fakeObservabilityLogger) Log(
	ctx context.Context,
	event ports.LogEvent,
) error {
	f.events = append(f.events, event)

	return f.err
}

func TestRequestObservabilityMiddleware_LogsSuccessfulRequest(t *testing.T) {
	logger := &fakeObservabilityLogger{}

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.NewRequestObservabilityMiddleware(
		logger,
	).Middleware(next)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/me",
		nil,
	)

	identity := ports.AuthenticatedIdentity{
		UserID: "user-123",
		Role:   "vendor",
	}

	request = request.WithContext(
		middleware.WithAuthenticatedIdentity(
			request.Context(),
			identity,
		),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status 200, got %d",
			recorder.Code,
		)
	}

	if len(logger.events) != 1 {
		t.Fatalf(
			"expected 1 log event, got %d",
			len(logger.events),
		)
	}

	event := logger.events[0]

	if event.Event != "http.request.completed" {
		t.Fatalf(
			"expected event http.request.completed, got %q",
			event.Event,
		)
	}

	if event.Operation != "/api/v1/users/me" {
		t.Fatalf(
			"expected operation /api/v1/users/me, got %q",
			event.Operation,
		)
	}

	if event.UserID != "user-123" {
		t.Fatalf(
			"expected user ID user-123, got %q",
			event.UserID,
		)
	}

	if event.Role != "vendor" {
		t.Fatalf(
			"expected role vendor, got %q",
			event.Role,
		)
	}

	if event.HTTPMethod != http.MethodGet {
		t.Fatalf(
			"expected method GET, got %q",
			event.HTTPMethod,
		)
	}

	if event.Route != "/api/v1/users/me" {
		t.Fatalf(
			"expected route /api/v1/users/me, got %q",
			event.Route,
		)
	}

	if event.StatusCode != http.StatusOK {
		t.Fatalf(
			"expected status code 200, got %d",
			event.StatusCode,
		)
	}

	if event.DurationMillis < 0 {
		t.Fatalf(
			"expected non-negative duration, got %d",
			event.DurationMillis,
		)
	}

	if event.FailureCategory != "" {
		t.Fatalf(
			"expected empty failure category, got %q",
			event.FailureCategory,
		)
	}
}

func TestRequestObservabilityMiddleware_LogsClientFailure(t *testing.T) {
	logger := &fakeObservabilityLogger{}

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.WriteHeader(http.StatusBadRequest)
	})

	handler := middleware.NewRequestObservabilityMiddleware(
		logger,
	).Middleware(next)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/register",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status 400, got %d",
			recorder.Code,
		)
	}

	if len(logger.events) != 1 {
		t.Fatalf(
			"expected 1 log event, got %d",
			len(logger.events),
		)
	}

	event := logger.events[0]

	if event.Event != "http.request.completed" {
		t.Fatalf(
			"expected event http.request.completed, got %q",
			event.Event,
		)
	}

	if event.StatusCode != http.StatusBadRequest {
		t.Fatalf(
			"expected status code 400, got %d",
			event.StatusCode,
		)
	}

	if event.FailureCategory != "client_error" {
		t.Fatalf(
			"expected failure category client_error, got %q",
			event.FailureCategory,
		)
	}

	if event.UserID != "" {
		t.Fatalf(
			"expected empty user ID, got %q",
			event.UserID,
		)
	}

	if event.Role != "" {
		t.Fatalf(
			"expected empty role, got %q",
			event.Role,
		)
	}
}

func TestRequestObservabilityMiddleware_LogsServerFailure(t *testing.T) {
	logger := &fakeObservabilityLogger{}

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	handler := middleware.NewRequestObservabilityMiddleware(
		logger,
	).Middleware(next)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/admin/users",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status 500, got %d",
			recorder.Code,
		)
	}

	if len(logger.events) != 1 {
		t.Fatalf(
			"expected 1 log event, got %d",
			len(logger.events),
		)
	}

	event := logger.events[0]

	if event.StatusCode != http.StatusInternalServerError {
		t.Fatalf(
			"expected status code 500, got %d",
			event.StatusCode,
		)
	}

	if event.FailureCategory != "server_error" {
		t.Fatalf(
			"expected failure category server_error, got %q",
			event.FailureCategory,
		)
	}
}

func TestRequestObservabilityMiddleware_UsesFallbackRouteWhenPatternUnavailable(
	t *testing.T,
) {
	logger := &fakeObservabilityLogger{}

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.WriteHeader(http.StatusNoContent)
	})

	handler := middleware.NewRequestObservabilityMiddleware(
		logger,
	).Middleware(next)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/logout",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if len(logger.events) != 1 {
		t.Fatalf(
			"expected 1 log event, got %d",
			len(logger.events),
		)
	}

	event := logger.events[0]

	if event.Route != "/api/v1/users/logout" {
		t.Fatalf(
			"expected fallback route /api/v1/users/logout, got %q",
			event.Route,
		)
	}

	if event.Operation != "/api/v1/users/logout" {
		t.Fatalf(
			"expected fallback operation /api/v1/users/logout, got %q",
			event.Operation,
		)
	}
}

func TestRequestObservabilityMiddleware_LoggerFailureDoesNotAffectResponse(
	t *testing.T,
) {
	logger := &fakeObservabilityLogger{
		err: context.Canceled,
	}

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.WriteHeader(http.StatusCreated)
	})

	handler := middleware.NewRequestObservabilityMiddleware(
		logger,
	).Middleware(next)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/register",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected status 201 despite logger failure, got %d",
			recorder.Code,
		)
	}

	if len(logger.events) != 1 {
		t.Fatalf(
			"expected 1 attempted log event, got %d",
			len(logger.events),
		)
	}
}

func TestRequestObservabilityMiddleware_NeverLogsSensitiveRequestData(
	t *testing.T,
) {
	logger := &fakeObservabilityLogger{}

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.NewRequestObservabilityMiddleware(
		logger,
	).Middleware(next)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/login",
		nil,
	)

	request.Header.Set(
		"Authorization",
		"Bearer-super-secret-token",
	)

	request = request.WithContext(
		middleware.WithAuthenticatedIdentity(
			request.Context(),
			ports.AuthenticatedIdentity{
				UserID: "user-123",
				Role:   "user",
			},
		),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if len(logger.events) != 1 {
		t.Fatalf(
			"expected 1 log event, got %d",
			len(logger.events),
		)
	}

	event := logger.events[0]

	if event.UserID == "Bearer-super-secret-token" ||
		event.Role == "Bearer-super-secret-token" ||
		event.Operation == "Bearer-super-secret-token" ||
		event.Route == "Bearer-super-secret-token" {
		t.Fatal("sensitive authorization data must never be logged")
	}

	if event.UserID == "password" ||
		event.Role == "password" ||
		event.Operation == "password" ||
		event.Route == "password" {
		t.Fatal("password data must never be logged")
	}
}
