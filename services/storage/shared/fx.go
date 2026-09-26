package shared

import (
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/shared/config"
)

// Module provides shared Storage dependencies.
var Module = fx.Module(
	"shared",

	fx.Provide(
		func() (*config.Config, error) {
			return config.Load()
		},
	),
)
