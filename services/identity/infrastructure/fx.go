package infrastructure

import (
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/persistence/postgres"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/security"
)

// Module provides infrastructure implementations required by the Identity
// service.
var Module = fx.Module(
	"infrastructure",

	fx.Provide(
		postgres.NewPool,
		postgres.NewQueries,

		fx.Annotate(
			postgres.NewUserRepository,
			fx.As(new(ports.UserRepository)),
		),

		fx.Annotate(
			security.NewBcryptPasswordHasher,
			fx.As(new(ports.PasswordHasher)),
		),
	),

	fx.Invoke(
		postgres.RegisterPoolLifecycle,
	),
)
