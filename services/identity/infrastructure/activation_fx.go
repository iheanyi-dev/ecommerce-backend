package infrastructure

import (
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/email"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/persistence/postgres"
)

var activationInfrastructureModule = fx.Provide(
	fx.Annotate(
		postgres.NewActivationTokenRepository,
		fx.As(new(ports.ActivationTokenRepository)),
	),
	fx.Annotate(
		email.NewSMTPSender,
		fx.As(new(ports.ActivationEmailSender)),
	),
)
