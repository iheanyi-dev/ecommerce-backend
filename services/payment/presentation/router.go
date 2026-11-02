package presentation

import (
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/presentation/handlers"
)

type MiddlewareSet struct {
	RequestID             func(http.Handler) http.Handler
	Tracing               func(http.Handler) http.Handler
	Observability         func(http.Handler) http.Handler
	ServiceAuthentication func(http.Handler) http.Handler
	IdentityPropagation   func(http.Handler) http.Handler
	RequireAuthentication func(http.Handler) http.Handler
}

func apply(handler http.Handler, middleware ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		if middleware[i] != nil {
			handler = middleware[i](handler)
		}
	}
	return handler
}

// NewRouter keeps public webhooks, Gateway requests, and trusted internal
// requests on separate middleware chains.
func NewRouter(h *handlers.Handlers, m MiddlewareSet, webhook http.Handler) http.Handler {
	mux := http.NewServeMux()

	internal := func(handler http.Handler) http.Handler {
		return apply(
			handler,
			m.RequestID,
			m.Tracing,
			m.ServiceAuthentication,
			m.Observability,
		)
	}

	gateway := func(handler http.Handler) http.Handler {
		return apply(
			handler,
			m.RequestID,
			m.Tracing,
			m.IdentityPropagation,
			m.Observability,
			m.RequireAuthentication,
		)
	}

	public := func(handler http.Handler) http.Handler {
		return apply(handler, m.RequestID, m.Tracing, m.Observability)
	}

	mux.Handle("POST /api/v1/payments", internal(h.InitiatePayment))
	mux.Handle("POST /api/v1/payments/{paymentID}/verify", internal(h.VerifyPayment))
	mux.Handle("GET /api/v1/payments/{paymentID}", gateway(h.GetPayment))
	mux.Handle("GET /api/v1/payments", gateway(h.ListPayments))

	mux.Handle("POST /api/v1/payouts", internal(h.InitiatePayout))
	mux.Handle("POST /api/v1/payouts/{payoutID}/verify", internal(h.VerifyPayout))
	mux.Handle("GET /api/v1/payouts/{payoutID}", gateway(h.GetPayout))
	mux.Handle("GET /api/v1/payouts", internal(h.ListPayouts))
	mux.Handle("POST /api/v1/payouts/{payoutID}/retry", internal(h.RetryPayout))

	if webhook != nil {
		mux.Handle("POST /api/v1/webhooks/paystack", public(webhook))
	}

	return mux
}
