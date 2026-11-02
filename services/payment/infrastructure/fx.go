package infrastructure

import (
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure/observability"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure/persistence/postgres"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure/providers/paystack"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/shared/config"
)

var Module = fx.Module(
	"payment-infrastructure",
	fx.Provide(
		config.Load,
		NewUseCases,
		func(useCases *UseCases) *use_cases.HandleProviderWebhookUseCase {
			return useCases.HandleWebhook
		},
		func(cfg *config.Config) (ports.Tracer, error) {
			return observability.NewOpenTelemetryTracer(cfg.ServiceAuthName)
		},
		func() (ports.Logger, error) {
			return observability.NewProductionLogger()
		},
		func() ports.Metrics {
			return observability.NewMetrics()
		},
	),
	postgres.Module,
	paystack.Module,
)
