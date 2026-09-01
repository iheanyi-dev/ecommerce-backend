package shared

import (
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/shared/config"
)

// Module provides shared application dependencies.
//
// Configuration is loaded once when the Fx application starts and is then
// made available to infrastructure and other layers through dependency
// injection.
var Module = fx.Module(
	"shared",

	fx.Provide(
		func() (*config.Config, error) {
			cfg, err := config.Load()
			if err != nil {
				return nil, err
			}

			return cfg, nil
		},
	),
)
