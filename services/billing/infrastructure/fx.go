package infrastructure

import (
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/infrastructure/messaging/kafka"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/infrastructure/observability"
	paymentclient "github.com/iheanyi-dev/ecommerce-backend/services/billing/infrastructure/payment"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/infrastructure/persistence/postgres"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/shared/config"
)

var Module = fx.Module(
	"billing-infrastructure",
	fx.Provide(
		config.Load,
		observability.NewBillingTracer,
		postgres.NewPool,
		fx.Annotate(postgres.NewUnitOfWork, fx.As(new(ports.UnitOfWork))),
		fx.Annotate(postgres.NewSubscriptionRepository, fx.As(new(ports.SubscriptionRepository))),
		fx.Annotate(postgres.NewPlanRepository, fx.As(new(ports.PlanRepository))),
		fx.Annotate(postgres.NewSubscriptionOutboxRepository, fx.As(new(ports.SubscriptionOutboxRepository))),
		fx.Annotate(kafka.NewProducer, fx.As(new(kafka.EventPublisher))),
		kafka.NewOutboxWorker,
		kafka.NewExpirationWorker,
		kafka.NewPaymentConsumer,
		fx.Annotate(paymentclient.NewClient, fx.As(new(ports.Payment))),
		use_cases.NewSubscribe,
		use_cases.NewUpdateSub,
		use_cases.NewExpireSubscriptions,
		use_cases.NewCancelSub,
		use_cases.NewCreatePlan,
		use_cases.NewUpdatePlan,
		use_cases.NewDeletePlanByID,
		use_cases.NewGetPlanByID,
		use_cases.NewGetPlans,
		use_cases.NewGetSubByID,
		use_cases.NewGetSubsByStoreID,
	),
	fx.Invoke(
		postgres.RegisterMigrations,
		func(*kafka.OutboxWorker, *kafka.PaymentConsumer, *kafka.ExpirationWorker) {},
	),
)
