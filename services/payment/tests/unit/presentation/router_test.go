package presentation_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/presentation"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/presentation/handlers"
)

func TestNewRouter_RoutesAndMiddlewareChains(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	h := &handlers.Handlers{
		InitiatePayment: ok,
		VerifyPayment:   ok,
		GetPayment:      ok,
		ListPayments:    ok,
		InitiatePayout:  ok,
		VerifyPayout:    ok,
		GetPayout:       ok,
		ListPayouts:     ok,
		RetryPayout:     ok,
	}

	mark := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("X-Middleware-Chain", name)
				next.ServeHTTP(w, r)
			})
		}
	}

	router := presentation.NewRouter(h, presentation.MiddlewareSet{
		RequestID:             mark("request-id"),
		Tracing:               mark("tracing"),
		Observability:         mark("observability"),
		ServiceAuthentication: mark("service-auth"),
		IdentityPropagation:   mark("identity"),
		RequireAuthentication: mark("user-auth"),
	}, ok)

	tests := []struct {
		name   string
		method string
		path   string
		chain  string
	}{
		{"initiate payment", http.MethodPost, "/api/v1/payments", "request-id,tracing,service-auth,observability"},
		{"verify payment", http.MethodPost, "/api/v1/payments/pay-1/verify", "request-id,tracing,service-auth,observability"},
		{"get payment", http.MethodGet, "/api/v1/payments/pay-1", "request-id,tracing,identity,observability,user-auth"},
		{"list payments", http.MethodGet, "/api/v1/payments", "request-id,tracing,identity,observability,user-auth"},
		{"initiate payout", http.MethodPost, "/api/v1/payouts", "request-id,tracing,service-auth,observability"},
		{"verify payout", http.MethodPost, "/api/v1/payouts/pay-1/verify", "request-id,tracing,service-auth,observability"},
		{"get payout", http.MethodGet, "/api/v1/payouts/pay-1", "request-id,tracing,identity,observability,user-auth"},
		{"list payouts", http.MethodGet, "/api/v1/payouts", "request-id,tracing,service-auth,observability"},
		{"retry payout", http.MethodPost, "/api/v1/payouts/pay-1/retry", "request-id,tracing,service-auth,observability"},
		{"Paystack webhook", http.MethodPost, "/api/v1/webhooks/paystack", "request-id,tracing,observability"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusNoContent {
				t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
			}

			got := strings.Join(rec.Header().Values("X-Middleware-Chain"), ",")
			if got != tt.chain {
				t.Fatalf("expected middleware chain %q, got %q", tt.chain, got)
			}
		})
	}
}

func TestNewRouter_WebhookCanBeOmitted(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	h := &handlers.Handlers{
		InitiatePayment: ok,
		VerifyPayment:   ok,
		GetPayment:      ok,
		ListPayments:    ok,
		InitiatePayout:  ok,
		VerifyPayout:    ok,
		GetPayout:       ok,
		ListPayouts:     ok,
		RetryPayout:     ok,
	}

	router := presentation.NewRouter(h, presentation.MiddlewareSet{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/paystack", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}
