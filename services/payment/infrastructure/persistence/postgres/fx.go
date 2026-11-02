package postgres

import (
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"payment-postgres",
	fx.Provide(
		NewPool,
		NewConfiguredBeneficiaryEncryptor,
		fx.Annotate(
			NewUnitOfWork,
			fx.As(new(ports.UnitOfWork)),
		),
		fx.Annotate(
			NewPaymentRepository,
			fx.As(new(ports.PaymentRepository)),
		),
		fx.Annotate(
			NewPaymentAttemptRepository,
			fx.As(new(ports.PaymentAttemptRepository)),
		),
		fx.Annotate(
			NewPayoutRepository,
			fx.As(new(ports.PayoutRepository)),
		),
		fx.Annotate(
			NewPayoutAttemptRepository,
			fx.As(new(ports.PayoutAttemptRepository)),
		),
	),
	fx.Invoke(RegisterMigrations),
)
