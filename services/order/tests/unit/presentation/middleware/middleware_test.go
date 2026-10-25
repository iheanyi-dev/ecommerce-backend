package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/infrastructure/observability"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/presentation/middleware"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/shared/config"
)

type loggerStub struct {
	events []ports.LogEvent
}

func (s *loggerStub) Log(_ context.Context, event ports.LogEvent) error {
	s.events = append(s.events, event)
	return nil
}

type metricsStub struct {
	increments []ports.Metric
	observes   []ports.Metric
}

func (s *metricsStub) Increment(_ context.Context, metric ports.Metric) error {
	s.increments = append(s.increments, metric)
	return nil
}

func (s *metricsStub) Observe(_ context.Context, metric ports.Metric) error {
	s.observes = append(s.observes, metric)
	return nil
}

type spanStub struct {
	ended bool
}

func (s *spanStub) End() { s.ended = true }

type tracerStub struct {
	name  string
	span  *spanStub
	key   any
	value any
}

func (s *tracerStub) Start(ctx context.Context, name string) (context.Context, ports.Span) {
	s.name = name
	ctx = context.WithValue(ctx, s.key, s.value)
	return ctx, s.span
}

func TestIdentityPropagationAcceptsValidIdentity(t *testing.T) {
	userID := uuid.New()
	handler := middleware.NewIdentityPropagationMiddleware().Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := ports.AuthenticatedIdentityFromContext(r.Context())
			if !ok || identity.UserID != userID || identity.Role != "customer" {
				t.Errorf("unexpected identity: %+v, present=%v", identity, ok)
			}
			w.WriteHeader(http.StatusNoContent)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(middleware.AuthenticatedUserIDHeader, userID.String())
	req.Header.Set(middleware.AuthenticatedRoleHeader, " customer ")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

func TestIdentityPropagationRejectsPartialOrMalformedIdentity(t *testing.T) {
	for _, tc := range []struct {
		name   string
		userID string
		role   string
	}{
		{name: "missing user ID", role: "customer"},
		{name: "missing role", userID: uuid.NewString()},
		{name: "malformed user ID", userID: "not-a-uuid", role: "customer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			handler := middleware.NewIdentityPropagationMiddleware().Middleware(
				http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
					called = true
				}),
			)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.userID != "" {
				req.Header.Set(middleware.AuthenticatedUserIDHeader, tc.userID)
			}
			if tc.role != "" {
				req.Header.Set(middleware.AuthenticatedRoleHeader, tc.role)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized || called {
				t.Fatalf("status=%d, handler called=%v; want 401 and not called", rec.Code, called)
			}
		})
	}
}

func TestIdentityPropagationAllowsAnonymousRequest(t *testing.T) {
	handler := middleware.NewIdentityPropagationMiddleware().Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := ports.AuthenticatedIdentityFromContext(r.Context()); ok {
				t.Error("unexpected identity on anonymous request")
			}
			w.WriteHeader(http.StatusNoContent)
		}),
	)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

func TestRequireAuthentication(t *testing.T) {
	called := false
	protected := middleware.RequireAuthentication(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))

	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnauthorized || called {
		t.Fatalf("unauthenticated status=%d, called=%v", rec.Code, called)
	}

	ctx := ports.WithAuthenticatedIdentity(context.Background(), ports.AuthenticatedIdentity{
		UserID: uuid.New(),
		Role:   "customer",
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	rec = httptest.NewRecorder()
	protected.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !called {
		t.Fatalf("authenticated status=%d, called=%v", rec.Code, called)
	}
}

func TestServiceAuthentication(t *testing.T) {
	cfg := &config.Config{
		OrderServiceAuthName:   "order",
		OrderServiceAuthSecret: "long-test-secret",
	}
	called := false
	handler := middleware.NewServiceAuthenticationMiddleware(cfg).Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusNoContent)
		}),
	)

	valid := httptest.NewRequest(http.MethodGet, "/", nil)
	valid.Header.Set(middleware.ServiceNameHeader, "order")
	valid.Header.Set(middleware.ServiceSecretHeader, "long-test-secret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, valid)
	if rec.Code != http.StatusNoContent || !called {
		t.Fatalf("valid credentials status=%d, called=%v", rec.Code, called)
	}

	for _, tc := range []struct {
		name    string
		service string
		secret  string
	}{
		{name: "wrong service", service: "store", secret: "long-test-secret"},
		{name: "wrong secret", service: "order", secret: "wrong"},
		{name: "missing secret", service: "order"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called = false
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set(middleware.ServiceNameHeader, tc.service)
			req.Header.Set(middleware.ServiceSecretHeader, tc.secret)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized || called {
				t.Fatalf("status=%d, called=%v; want 401 and not called", rec.Code, called)
			}
		})
	}
}

func TestServiceAuthenticationFailsClosedWithoutConfiguration(t *testing.T) {
	handler := middleware.NewServiceAuthenticationMiddleware(nil).Middleware(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Fatal("handler must not be called")
		}),
	)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestRequestIDPreservesValidUUIDAndGeneratesForInvalid(t *testing.T) {
	for _, supplied := range []string{uuid.NewString(), "invalid"} {
		t.Run(supplied, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set(middleware.RequestIDHeader, supplied)
			rec := httptest.NewRecorder()
			middleware.RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get(middleware.RequestIDHeader) != w.Header().Get(middleware.RequestIDHeader) {
					t.Error("request and response request IDs differ")
				}
			})).ServeHTTP(rec, req)

			got := rec.Header().Get(middleware.RequestIDHeader)
			if _, err := uuid.Parse(got); err != nil {
				t.Fatalf("request ID %q is not a UUID: %v", got, err)
			}
			if supplied != "invalid" && got != supplied {
				t.Fatalf("request ID = %q, want preserved %q", got, supplied)
			}
		})
	}
}

func TestObservabilityRecordsStatusAndMetrics(t *testing.T) {
	logger := &loggerStub{}
	metrics := &metricsStub{}
	userID := uuid.New()
	ctx := ports.WithAuthenticatedIdentity(context.Background(), ports.AuthenticatedIdentity{
		UserID: userID,
		Role:   "customer",
	})

	handler := middleware.NewRequestObservabilityMiddleware(logger, metrics).Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "failure", http.StatusInternalServerError)
		}),
	)
	req := httptest.NewRequest(http.MethodPost, "/orders", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if len(logger.events) != 1 || logger.events[0].Event != "http.request.completed" {
		t.Fatalf("unexpected log events: %+v", logger.events)
	}
	if logger.events[0].FailureCategory != "server_error" ||
		logger.events[0].UserID != userID.String() ||
		logger.events[0].Role != "customer" {
		t.Fatalf("unexpected log event: %+v", logger.events[0])
	}
	if len(metrics.increments) != 1 || metrics.increments[0].Name != "http.requests" {
		t.Fatalf("unexpected increment metrics: %+v", metrics.increments)
	}
	if metrics.increments[0].Labels["status"] != "500" {
		t.Fatalf("unexpected metric labels: %+v", metrics.increments[0].Labels)
	}
	if len(metrics.observes) != 1 || metrics.observes[0].Name != "http.request.duration" {
		t.Fatalf("unexpected duration metrics: %+v", metrics.observes)
	}
}

func TestTracingMiddlewareStartsEndsSpanAndPropagatesContext(t *testing.T) {
	type contextKey string
	key := contextKey("trace-test")
	span := &spanStub{}
	tracer := &tracerStub{name: "", span: span, key: key, value: "traced"}

	handler := middleware.NewTracingMiddleware(tracer).Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Context().Value(key); got != "traced" {
				t.Errorf("context value = %v, want traced", got)
			}
			w.WriteHeader(http.StatusNoContent)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	req.Pattern = "GET /orders"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if tracer.name != "GET /orders" {
		t.Fatalf("span name = %q, want %q", tracer.name, "GET /orders")
	}
	if !span.ended {
		t.Fatal("span was not ended")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

func TestTracingMiddlewareWithoutTracerPassesThrough(t *testing.T) {
	called := false
	handler := middleware.NewTracingMiddleware(nil).Middleware(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			called = true
		}),
	)
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if !called {
		t.Fatal("handler was not called")
	}
}

func TestOpenTelemetryTracerRejectsEmptyName(t *testing.T) {
	_, err := observability.NewOpenTelemetryTracer("   ")
	if err == nil {
		t.Fatal("expected error for empty tracer name")
	}
}

func TestOpenTelemetryTracerCanStartAndEndSpan(t *testing.T) {
	tracer, err := observability.NewOpenTelemetryTracer("order-test")
	if err != nil {
		t.Fatal(err)
	}
	ctx, span := tracer.Start(context.Background(), "GET /orders")
	if ctx == nil || span == nil {
		t.Fatal("expected context and span")
	}
	span.End()

	_, noop := tracer.Start(context.Background(), " ")
	if noop == nil {
		t.Fatal("expected no-op span for empty span name")
	}
	noop.End()
}

func TestResponseRecorderDoesNotReplaceFirstStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	handler := middleware.NewRequestObservabilityMiddleware(nil, nil).Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusAccepted)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("ok"))
		}),
	)
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", rec.Code)
	}
	if strings.TrimSpace(rec.Body.String()) != "ok" {
		t.Fatalf("body = %q, want ok", rec.Body.String())
	}
}
