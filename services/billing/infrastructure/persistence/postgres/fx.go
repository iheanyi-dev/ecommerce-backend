package postgres

import (
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"billing-postgres",
	fx.Provide(
		NewPool,
		fx.Annotate(NewUnitOfWork, fx.As(new(ports.UnitOfWork))),
		fx.Annotate(NewSubscriptionRepository, fx.As(new(ports.SubscriptionRepository))),
		fx.Annotate(NewSubscriptionOutboxRepository, fx.As(new(ports.SubscriptionOutboxRepository))),
	),
	fx.Invoke(RegisterMigrations),
)
