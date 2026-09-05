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
		// PostgreSQL connection pool.
		postgres.NewPool,

		// SQLC query implementation.
		postgres.NewQueries,

		// User repository implementation exposed through the
		// application-layer UserRepository port.
		fx.Annotate(
			postgres.NewUserRepository,
			fx.As(new(ports.UserRepository)),
		),

		// Refresh-token repository implementation exposed through
		// the application-layer RefreshTokenRepository port.
		fx.Annotate(
			postgres.NewRefreshTokenRepository,
			fx.As(new(ports.RefreshTokenRepository)),
		),

		// Bcrypt password hashing implementation exposed through
		// the application-layer PasswordHasher port.
		fx.Annotate(
			security.NewBcryptPasswordHasher,
			fx.As(new(ports.PasswordHasher)),
		),

		// JWT access-token implementation exposed through the
		// application-layer TokenService port.
		fx.Annotate(
			security.NewJWTTokenServiceFromConfig,
			fx.As(new(ports.TokenService)),
		),

		// Refresh-token implementation exposed through the
		// application-layer RefreshTokenService port.
		fx.Annotate(
			security.NewRefreshTokenService,
			fx.As(new(ports.RefreshTokenService)),
		),
	),

	fx.Invoke(
		postgres.RegisterPoolLifecycle,
	),
)
