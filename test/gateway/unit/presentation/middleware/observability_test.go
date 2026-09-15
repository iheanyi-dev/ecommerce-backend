package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/gateway/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/gateway/presentation/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLogger struct {
	events []logEvent
}

type logEvent struct {
	message string
	fields  []ports.Field
}

func (f *fakeLogger) Info(
	_ context.Context,
	message string,
	fields ...ports.Field,
) {
	f.events = append(f.events, logEvent{
		message: message,
		fields:  fields,
	})
}

func (f *fakeLogger) Error(
	_ context.Context,
	message string,
	fields ...ports.Field,
) {
	f.events = append(f.events, logEvent{
		message: message,
		fields:  fields,
	})
}

func TestRequestObservabilityMiddleware_LogsCompletedRequest(t *testing.T) {
	logger := &fakeLogger{}

	handler := middleware.NewRequestObservabilityMiddleware(logger).
		Middleware(
			http.HandlerFunc(func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				w.WriteHeader(http.StatusCreated)
			}),
		)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/register",
		nil,
	)
	request.Header.Set(
		middleware.RequestIDHeader,
		"11111111-1111-1111-1111-111111111111",
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusCreated, recorder.Code)
	require.Len(t, logger.events, 1)

	event := logger.events[0]

	assert.Equal(t, "http.request.completed", event.message)

	assertField(t, event.fields, "http.method", http.MethodPost)
	assertField(t, event.fields, "http.path", "/api/v1/users/register")
	assertField(t, event.fields, "http.status_code", http.StatusCreated)
	assertField(
		t,
		event.fields,
		"request_id",
		"11111111-1111-1111-1111-111111111111",
	)

	assertFieldExists(t, event.fields, "http.duration_ms")
}

func TestRequestObservabilityMiddleware_LogsImplicitOKStatus(t *testing.T) {
	logger := &fakeLogger{}

	handler := middleware.NewRequestObservabilityMiddleware(logger).
		Middleware(
			http.HandlerFunc(func(
				w http.ResponseWriter,
				_ *http.Request,
			) {
				_, err := w.Write([]byte(`{"status":"ok"}`))
				require.NoError(t, err)
			}),
		)

	request := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Len(t, logger.events, 1)

	assertField(
		t,
		logger.events[0].fields,
		"http.status_code",
		http.StatusOK,
	)
}

func TestRequestObservabilityMiddleware_DoesNotAlterResponse(t *testing.T) {
	logger := &fakeLogger{}

	handler := middleware.NewRequestObservabilityMiddleware(logger).
		Middleware(
			http.HandlerFunc(func(
				w http.ResponseWriter,
				_ *http.Request,
			) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadGateway)

				_, err := w.Write([]byte(`{"error":"downstream unavailable"}`))
				require.NoError(t, err)
			}),
		)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/me",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusBadGateway, recorder.Code)
	assert.Equal(
		t,
		"application/json",
		recorder.Header().Get("Content-Type"),
	)
	assert.Equal(
		t,
		`{"error":"downstream unavailable"}`,
		recorder.Body.String(),
	)
}

func assertField(
	t *testing.T,
	fields []ports.Field,
	key string,
	expected any,
) {
	t.Helper()

	for _, field := range fields {
		if field.Key == key {
			assert.Equal(t, expected, field.Value)
			return
		}
	}

	require.Failf(
		t,
		"missing log field",
		"expected field %q",
		key,
	)
}

func assertFieldExists(
	t *testing.T,
	fields []ports.Field,
	key string,
) {
	t.Helper()

	for _, field := range fields {
		if field.Key == key {
			return
		}
	}

	require.Failf(
		t,
		"missing log field",
		"expected field %q",
		key,
	)
}
