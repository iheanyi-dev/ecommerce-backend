package application

import (
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/use_cases"
	"go.uber.org/fx"
)

// Module provides the Storage application use cases.
//
// Each use case depends on application ports. The concrete implementations
// of those ports are supplied by the infrastructure module.
var Module = fx.Module(
	"application",

	fx.Provide(
		use_cases.NewPutObjectUseCase,
		use_cases.NewGetPublicObjectUseCase,
		use_cases.NewGetPrivateObjectUseCase,
		use_cases.NewDeleteObjectUseCase,
	),
)
