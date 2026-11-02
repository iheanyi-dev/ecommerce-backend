package presentation_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/middleware"
	"github.com/stretchr/testify/require"
)

type testLogger struct {
	events []ports.LogEvent
}

func (l *testLogger) Log(ctx context.Context, event ports.LogEvent) error {
	l.events = append(l.events, event)
	return nil
}

type testMetrics struct {
	increments   []ports.Metric
	observations []ports.Metric
}

func (m *testMetrics) Increment(ctx context.Context, metric ports.Metric) error {
	m.increments = append(m.increments, metric)
	return nil
}

func (m *testMetrics) Observe(ctx context.Context, metric ports.Metric) error {
	m.observations = append(m.observations, metric)
	return nil
}

func TestRequestObservabilityMiddleware_RecordsCompletedRequest(t *testing.T) {
	logger := &testLogger{}
	metrics := &testMetrics{}

	m := middleware.NewRequestObservabilityMiddleware(
		logger,
		metrics,
	)

	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/stores",
		nil,
	)
	request.Pattern = "POST /api/v1/stores"

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusCreated, recorder.Code)

	require.Len(t, logger.events, 1)
	require.Equal(t, "http.request.completed", logger.events[0].Event)
	require.Equal(t, "POST /api/v1/stores", logger.events[0].Operation)
	require.Empty(t, logger.events[0].UserID)
	require.Empty(t, logger.events[0].Role)
	require.Empty(t, logger.events[0].FailureCategory)

	require.Len(t, metrics.increments, 1)
	require.Equal(t, "http.requests", metrics.increments[0].Name)
	require.Equal(t, float64(1), metrics.increments[0].Value)
	require.Equal(t, "POST", metrics.increments[0].Labels["method"])
	require.Equal(t, "POST /api/v1/stores", metrics.increments[0].Labels["route"])
	require.Equal(t, "201", metrics.increments[0].Labels["status"])

	require.Len(t, metrics.observations, 1)
	require.Equal(t, "http.request.duration", metrics.observations[0].Name)
	require.GreaterOrEqual(t, metrics.observations[0].Value, float64(0))
}

func TestRequestObservabilityMiddleware_RecordsAuthenticatedIdentity(t *testing.T) {
	logger := &testLogger{}

	m := middleware.NewRequestObservabilityMiddleware(
		logger,
		nil,
	)

	userID := uuid.New()

	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/stores/owner",
		nil,
	)

	request = request.WithContext(
		ports.WithAuthenticatedIdentity(
			request.Context(),
			ports.AuthenticatedIdentity{
				UserID: userID,
				Role:   "vendor",
			},
		),
	)

	request.Pattern = "GET /api/v1/stores/owner"

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	require.Len(t, logger.events, 1)
	require.Equal(t, userID.String(), logger.events[0].UserID)
	require.Equal(t, "vendor", logger.events[0].Role)
}

func TestRequestObservabilityMiddleware_RecordsFailureCategory(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		expectedReason string
	}{
		{
			name:           "client error",
			status:         http.StatusBadRequest,
			expectedReason: "client_error",
		},
		{
			name:           "server error",
			status:         http.StatusInternalServerError,
			expectedReason: "server_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &testLogger{}

			m := middleware.NewRequestObservabilityMiddleware(
				logger,
				nil,
			)

			handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
			}))

			request := httptest.NewRequest(
				http.MethodGet,
				"/api/v1/stores",
				nil,
			)
			request.Pattern = "GET /api/v1/stores"

			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			require.Len(t, logger.events, 1)
			require.Equal(
				t,
				tt.expectedReason,
				logger.events[0].FailureCategory,
			)
		})
	}
}

func TestRequestObservabilityMiddleware_ObservabilityFailuresDoNotBreakRequest(t *testing.T) {
	handler := middleware.NewRequestObservabilityMiddleware(
		nil,
		nil,
	)

	nextCalled := false

	wrapped := handler.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/stores",
		nil,
	)
	recorder := httptest.NewRecorder()

	wrapped.ServeHTTP(recorder, request)

	require.True(t, nextCalled)
	require.Equal(t, http.StatusNoContent, recorder.Code)
}

type flusherResponseWriter struct {
	*httptest.ResponseRecorder
	flushed bool
}

func (w *flusherResponseWriter) Flush() {
	w.flushed = true
	w.ResponseRecorder.Flush()
}

func TestRequestObservabilityMiddleware_PreservesHTTPFlusherForStreamingHandlers(t *testing.T) {
	logger := &testLogger{}

	m := middleware.NewRequestObservabilityMiddleware(
		logger,
		nil,
	)

	underlying := &flusherResponseWriter{
		ResponseRecorder: httptest.NewRecorder(),
	}

	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		require.True(t, ok, "observability middleware must preserve http.Flusher")

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte("data: test\n\n"))
		require.NoError(t, err)

		flusher.Flush()
	}))

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/stores/test/chat",
		nil,
	)
	request.Pattern = "POST /api/v1/stores/{id}/chat"

	handler.ServeHTTP(underlying, request)

	require.Equal(t, http.StatusOK, underlying.Code)
	require.True(t, underlying.flushed)
	require.Equal(t, "data: test\n\n", underlying.Body.String())

	require.Len(t, logger.events, 1)
	require.Equal(
		t,
		"POST /api/v1/stores/{id}/chat",
		logger.events[0].Operation,
	)
}
