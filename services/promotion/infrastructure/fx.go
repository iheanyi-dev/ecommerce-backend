package infrastructure

import (
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/infrastructure/messaging/kafka"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/infrastructure/observability"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/infrastructure/persistence/postgres"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/infrastructure/workers"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/shared/config"
	"go.uber.org/fx"
)

// Module provides the Promotion service infrastructure and background workers.
var Module = fx.Module(
	"promotion",

	fx.Provide(
		config.Load,
		func() (ports.Tracer, error) {
			return observability.NewOpenTelemetryTracer("promotion")
		},
		postgres.NewPool,
		postgres.NewPromotionRepository,
		postgres.NewPromotionOutboxRepository,
		postgres.NewUnitOfWork,

		func(repository *postgres.PromotionRepository) ports.PromotionRepository {
			return repository
		},
		func(repository *postgres.PromotionOutboxRepository) ports.PromotionOutboxRepository {
			return repository
		},
		func(unitOfWork *postgres.UnitOfWork) ports.UnitOfWork {
			return unitOfWork
		},
		func(
			lifecycle fx.Lifecycle,
			cfg *config.Config,
		) (ports.PromotionEventPublisher, error) {
			return kafka.NewProducer(
				lifecycle,
				cfg.KafkaBrokers,
				cfg.PromotionEventsTopic,
			)
		},

		use_cases.NewActivatePromotionUseCase,
		use_cases.NewGetPromotionByIDUseCase,
		use_cases.NewGetPromosForUserUseCase,
		use_cases.NewGetPromoForUserUseCase,
		use_cases.NewCreatePromotionUseCase,
		use_cases.NewCancelPromotionUseCase,
		use_cases.NewRenewPromotionUseCase,
		use_cases.NewDeletePromotionUseCase,
		use_cases.NewGetPromosByStoreUseCase,
		use_cases.NewExpirePromotionUseCase,
		workers.NewOutboxWorker,
	),

	// Register the scheduled activation and expiration worker.
	fx.Invoke(
		postgres.RegisterMigrations,
		func(*workers.OutboxWorker) {},

		func(
			lifecycle fx.Lifecycle,
			repository *postgres.PromotionRepository,
			activate *use_cases.ActivatePromotionUseCase,
			expire *use_cases.ExpirePromotionUseCase,
		) {
			workers.NewLifecycleWorker(
				lifecycle,
				repository,
				activate,
				expire,
			)
		},
	),
)
