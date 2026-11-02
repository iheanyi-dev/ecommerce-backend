package presentation

import (
	grpcserver "github.com/iheanyi-dev/ecommerce-backend/services/payment/presentation/grpc"
	"net/http"

	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/presentation/handlers"
	paymenthttp "github.com/iheanyi-dev/ecommerce-backend/services/payment/presentation/http"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/presentation/middleware"
)

var Module = fx.Module(
	"payment-presentation",
	fx.Provide(
		handlers.New,
		paymenthttp.NewProviderWebhookHandler,

		middleware.NewRequestIDMiddleware,
		middleware.NewTracingMiddleware,
		middleware.NewRequestObservabilityMiddleware,
		middleware.NewServiceAuthenticationMiddleware,
		middleware.NewIdentityPropagationMiddleware,

		NewPaymentRouter,
		grpcserver.NewServer,
	),
	fx.Invoke(grpcserver.RegisterGRPCServer),
)

func NewPaymentRouter(
	h *handlers.Handlers,
	webhook *paymenthttp.ProviderWebhookHandler,
	requestID *middleware.RequestIDMiddleware,
	tracing *middleware.TracingMiddleware,
	observability *middleware.RequestObservabilityMiddleware,
	serviceAuth *middleware.ServiceAuthenticationMiddleware,
	identity *middleware.IdentityPropagationMiddleware,
) http.Handler {
	return NewRouter(h, MiddlewareSet{
		RequestID:             requestID.Middleware,
		Tracing:               tracing.Middleware,
		Observability:         observability.Middleware,
		ServiceAuthentication: serviceAuth.RequireServiceAuthentication,
		IdentityPropagation:   identity.Middleware,
		RequireAuthentication: middleware.RequireAuthentication,
	}, webhook)
}
