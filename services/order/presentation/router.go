package presentation

import (
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/order/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/presentation/middleware"
)

// NewRouter builds the authenticated Order HTTP API.
func NewRouter(
	h *handlers.OrderHandlers,
	serviceAuth *middleware.ServiceAuthenticationMiddleware,
	tracing *middleware.TracingMiddleware,
	observability *middleware.RequestObservabilityMiddleware,
) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/orders", h.CreateOrder)
	mux.HandleFunc("GET /api/v1/orders", h.ListOrders)
	mux.HandleFunc("GET /api/v1/orders/{id}", h.GetOrder)

	mux.Handle("POST /api/v1/orders/{id}/shipping",
		middleware.RequireAuthentication(http.HandlerFunc(h.AddShipping)))
	mux.Handle("POST /api/v1/orders/{id}/shipping/reject",
		middleware.RequireAuthentication(http.HandlerFunc(h.RejectShipping)))
	mux.Handle("POST /api/v1/orders/{id}/cancel",
		middleware.RequireAuthentication(http.HandlerFunc(h.CancelOrder)))
	mux.Handle("POST /api/v1/orders/{id}/sent",
		middleware.RequireAuthentication(http.HandlerFunc(h.MarkSent)))
	mux.Handle("POST /api/v1/orders/{id}/received",
		middleware.RequireAuthentication(http.HandlerFunc(h.ConfirmReceipt)))
	mux.Handle("POST /api/v1/orders/{id}/payment",
		middleware.RequireAuthentication(http.HandlerFunc(h.InitiatePayment)))

	// Request ID -> service authentication -> tracing -> observability
	// -> identity propagation -> route authentication.
	var handler http.Handler = mux
	handler = middleware.NewIdentityPropagationMiddleware().Middleware(handler)

	if observability != nil {
		handler = observability.Middleware(handler)
	}
	if tracing != nil {
		handler = tracing.Middleware(handler)
	}
	if serviceAuth != nil {
		handler = serviceAuth.Middleware(handler)
	}
	handler = middleware.RequestIDMiddleware(handler)

	return handler
}
