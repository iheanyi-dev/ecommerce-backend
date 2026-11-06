package presentation

import (
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/middleware"
)

// NewRouter builds the Store HTTP routing tree.
//
// Service authentication protects the complete Store service boundary.
// Every request must originate from a trusted internal service.
//
// User authentication remains a separate route-level concern. Public Store
// discovery therefore remains anonymous to the end user while still requiring
// the calling Gateway or internal service to authenticate itself.
func NewRouter(
	createStoreHandler http.Handler,
	getStoreByIDHandler http.Handler,
	getStoreByOwnerHandler http.Handler,
	getStoreBySlugHandler http.Handler,
	listStoresHandler http.Handler,
	updateStoreHandler http.Handler,
	changeStoreImageHandler http.Handler,
	changeStorePlanHandler http.Handler,
	subscribeHandler http.Handler,
	listBillingPlansHandler http.Handler,
	getBillingPlanHandler http.Handler,
	listStoreSubscriptionsHandler http.Handler,
	getStoreSubscriptionHandler http.Handler,
	changeStoreStatusHandler http.Handler,
	addStoreKnowledgeHandler http.Handler,
	chatHandler http.Handler,
	removeStaleKnowledgeHandler http.Handler,
	listStoreOrdersHandler http.Handler,
	getStoreOrderHandler http.Handler,
	addShippingHandler http.Handler,
	markOrderSentHandler http.Handler,
	cancelStoreOrderHandler http.Handler,
	initiatePayoutHandler http.Handler,
	serviceAuthenticationMiddleware *middleware.ServiceAuthenticationMiddleware,
	requestObservabilityMiddleware *middleware.RequestObservabilityMiddleware,
	tracingMiddleware *middleware.TracingMiddleware,
	requestIDMiddleware *middleware.RequestIDMiddleware,
) http.Handler {
	mux := http.NewServeMux()

	// Public discovery.
	mux.Handle(
		"GET /api/v1/stores",
		listStoresHandler,
	)
	mux.Handle(
		"GET /api/v1/stores/plans",
		listBillingPlansHandler,
	)
	mux.Handle(
		"GET /api/v1/stores/plans/{id}",
		getBillingPlanHandler,
	)
	mux.Handle(
		"GET /api/v1/stores/{id}",
		getStoreByIDHandler,
	)
	mux.Handle(
		"GET /api/v1/stores/slug/{slug}",
		getStoreBySlugHandler,
	)

	// User-authenticated operations.
	mux.Handle(
		"POST /api/v1/stores",
		middleware.RequireAuthentication(createStoreHandler),
	)
	mux.Handle(
		"GET /api/v1/stores/owner",
		middleware.RequireAuthentication(getStoreByOwnerHandler),
	)
	mux.Handle(
		"PATCH /api/v1/stores/{id}",
		middleware.RequireAuthentication(updateStoreHandler),
	)
	mux.Handle(
		"PATCH /api/v1/stores/{id}/image",
		middleware.RequireAuthentication(changeStoreImageHandler),
	)
	mux.Handle(
		"PATCH /api/v1/stores/{id}/plan",
		middleware.RequireAuthentication(changeStorePlanHandler),
	)
	mux.Handle(
		"POST /api/v1/stores/{id}/subscribe",
		middleware.RequireAuthentication(subscribeHandler),
	)
	mux.Handle(
		"GET /api/v1/stores/{id}/billing/subscriptions",
		middleware.RequireAuthentication(listStoreSubscriptionsHandler),
	)
	mux.Handle(
		"GET /api/v1/stores/{id}/billing/subscriptions/{subscriptionID}",
		middleware.RequireAuthentication(getStoreSubscriptionHandler),
	)
	mux.Handle(
		"POST /api/v1/stores/{id}/knowledge",
		middleware.RequireAuthentication(addStoreKnowledgeHandler),
	)
	mux.Handle(
		"POST /api/v1/stores/{id}/chat",
		middleware.RequireAuthentication(chatHandler),
	)
	mux.Handle(
		"DELETE /api/v1/stores/{id}/knowledge/stale",
		middleware.RequireAuthentication(removeStaleKnowledgeHandler),
	)

	mux.Handle("GET /api/v1/stores/owner/orders", middleware.RequireAuthentication(listStoreOrdersHandler))
	mux.Handle("GET /api/v1/stores/owner/orders/{orderID}", middleware.RequireAuthentication(getStoreOrderHandler))
	mux.Handle("POST /api/v1/stores/owner/orders/{orderID}/shipping", middleware.RequireAuthentication(addShippingHandler))
	mux.Handle("POST /api/v1/stores/owner/orders/{orderID}/sent", middleware.RequireAuthentication(markOrderSentHandler))
	mux.Handle("POST /api/v1/stores/owner/orders/{orderID}/cancel", middleware.RequireAuthentication(cancelStoreOrderHandler))
	mux.Handle("POST /api/v1/stores/owner/payouts", middleware.RequireAuthentication(initiatePayoutHandler))

	// Subscription/Billing-controlled lifecycle operation.
	//
	// The entire Store service is already protected by service
	// authentication below. This endpoint therefore does not need a second
	// service-authentication wrapper.
	mux.Handle(
		"PATCH /api/v1/stores/{id}/status",
		changeStoreStatusHandler,
	)

	// Compose the Store service boundary.
	//
	// The order is deliberately:
	//
	//	Request ID
	//	    ↓
	//	Service Authentication
	//	    ↓
	//	Tracing
	//	    ↓
	//	HTTP Observability
	//	    ↓
	//	Identity Propagation
	//	    ↓
	//	Router
	//	    ↓
	//	Route authentication
	//
	// Request ID is outermost so even rejected service requests receive a
	// correlation identifier.
	var handler http.Handler = mux

	handler = middleware.NewIdentityPropagationMiddleware().Middleware(handler)

	if requestObservabilityMiddleware != nil {
		handler = requestObservabilityMiddleware.Middleware(handler)
	}

	if tracingMiddleware != nil {
		handler = tracingMiddleware.Middleware(handler)
	}

	if serviceAuthenticationMiddleware != nil {
		handler = serviceAuthenticationMiddleware.RequireServiceAuthentication(handler)
	}

	if requestIDMiddleware != nil {
		handler = requestIDMiddleware.Middleware(handler)
	}

	return handler
}
