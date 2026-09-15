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

// fakeObservabilityMetrics captures metrics emitted by the HTTP
// observability middleware.
//
// The fake deliberately stores the application-level Metric values rather
// than depending on the concrete infrastructure implementation. This keeps
// the middleware tests focused on the behavior required from the Metrics
// port.
type fakeObservabilityMetrics struct {
	increments   []ports.Metric
	observations []ports.Metric
	err          error
}

func (f *fakeObservabilityMetrics) Increment(
	ctx context.Context,
	metric ports.Metric,
) error {
	f.increments = append(f.increments, metric)

	return f.err
}

func (f *fakeObservabilityMetrics) Observe(
	ctx context.Context,
	metric ports.Metric,
) error {
	f.observations = append(f.observations, metric)

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
		nil,
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
		nil,
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
		nil,
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
		nil,
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
		nil,
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
		nil,
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

// TestRequestObservabilityMiddleware_RecordsHTTPMetrics verifies that one
// completed HTTP request produces both the request counter and a duration
// measurement.
//
// This is the core transport-level metric behavior. We do not duplicate every
// logging test with metric assertions because status classification and route
// selection are already covered by the logging tests.
func TestRequestObservabilityMiddleware_RecordsHTTPMetrics(t *testing.T) {
	logger := &fakeObservabilityLogger{}
	metrics := &fakeObservabilityMetrics{}

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.NewRequestObservabilityMiddleware(
		logger,
		metrics,
	).Middleware(next)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/me",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status 200, got %d",
			recorder.Code,
		)
	}

	if len(metrics.increments) != 1 {
		t.Fatalf(
			"expected 1 metric increment, got %d",
			len(metrics.increments),
		)
	}

	requestMetric := metrics.increments[0]

	if requestMetric.Name != "http.requests" {
		t.Fatalf(
			"expected metric http.requests, got %q",
			requestMetric.Name,
		)
	}

	if requestMetric.Value != 1 {
		t.Fatalf(
			"expected request metric value 1, got %f",
			requestMetric.Value,
		)
	}

	if requestMetric.Labels["method"] != http.MethodGet {
		t.Fatalf(
			"expected method label GET, got %q",
			requestMetric.Labels["method"],
		)
	}

	if requestMetric.Labels["route"] != "/api/v1/users/me" {
		t.Fatalf(
			"expected route label /api/v1/users/me, got %q",
			requestMetric.Labels["route"],
		)
	}

	if requestMetric.Labels["status"] != "200" {
		t.Fatalf(
			"expected status label 200, got %q",
			requestMetric.Labels["status"],
		)
	}

	if len(metrics.observations) != 1 {
		t.Fatalf(
			"expected 1 metric observation, got %d",
			len(metrics.observations),
		)
	}

	durationMetric := metrics.observations[0]

	if durationMetric.Name != "http.request.duration" {
		t.Fatalf(
			"expected metric http.request.duration, got %q",
			durationMetric.Name,
		)
	}

	if durationMetric.Value < 0 {
		t.Fatalf(
			"expected non-negative duration, got %f",
			durationMetric.Value,
		)
	}

	if durationMetric.Labels["method"] != http.MethodGet {
		t.Fatalf(
			"expected duration method label GET, got %q",
			durationMetric.Labels["method"],
		)
	}

	if durationMetric.Labels["route"] != "/api/v1/users/me" {
		t.Fatalf(
			"expected duration route label /api/v1/users/me, got %q",
			durationMetric.Labels["route"],
		)
	}
}

// TestRequestObservabilityMiddleware_MetricsFailureDoesNotAffectResponse
// verifies that metrics are best-effort just like logging.
//
// An observability backend failure must never turn an otherwise successful
// application request into an HTTP failure.
func TestRequestObservabilityMiddleware_MetricsFailureDoesNotAffectResponse(
	t *testing.T,
) {
	logger := &fakeObservabilityLogger{}
	metrics := &fakeObservabilityMetrics{
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
		metrics,
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
			"expected status 201 despite metrics failure, got %d",
			recorder.Code,
		)
	}

	if len(metrics.increments) != 1 {
		t.Fatalf(
			"expected 1 attempted metric increment, got %d",
			len(metrics.increments),
		)
	}

	if len(metrics.observations) != 1 {
		t.Fatalf(
			"expected 1 attempted metric observation, got %d",
			len(metrics.observations),
		)
	}
}

// TestRequestObservabilityMiddleware_UsesStableMetricLabels verifies that
// metrics use stable transport dimensions rather than sensitive or
// high-cardinality request data.
//
// In particular, the request path is the route fallback here, so this test
// also protects against accidentally putting query strings or request bodies
// into metric labels.
func TestRequestObservabilityMiddleware_UsesStableMetricLabels(t *testing.T) {
	logger := &fakeObservabilityLogger{}
	metrics := &fakeObservabilityMetrics{}

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.WriteHeader(http.StatusBadRequest)
	})

	handler := middleware.NewRequestObservabilityMiddleware(
		logger,
		metrics,
	).Middleware(next)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/login?email=secret@example.com",
		nil,
	)

	request.Header.Set(
		"Authorization",
		"Bearer-super-secret-token",
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if len(metrics.increments) != 1 {
		t.Fatalf(
			"expected 1 metric increment, got %d",
			len(metrics.increments),
		)
	}

	metric := metrics.increments[0]

	if metric.Labels["method"] != http.MethodPost {
		t.Fatalf(
			"expected method label POST, got %q",
			metric.Labels["method"],
		)
	}

	if metric.Labels["route"] != "/api/v1/users/login" {
		t.Fatalf(
			"expected stable route label without query string, got %q",
			metric.Labels["route"],
		)
	}

	if metric.Labels["status"] != "400" {
		t.Fatalf(
			"expected status label 400, got %q",
			metric.Labels["status"],
		)
	}

	for key, value := range metric.Labels {
		if value == "secret@example.com" ||
			value == "Bearer-super-secret-token" {
			t.Fatalf(
				"sensitive request data must never appear in metric label %q=%q",
				key,
				value,
			)
		}
	}
}
