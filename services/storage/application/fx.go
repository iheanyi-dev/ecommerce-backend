package application

import (
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/use_cases"
	"go.uber.org/fx"
)

// Module provides the Storage application use cases.
//
// Each use case is exposed through its application-layer port so adapters
// depend on stable application contracts rather than concrete use cases.
var Module = fx.Module(
	"application",

	fx.Provide(
		fx.Annotate(
			use_cases.NewPutObjectUseCase,
			fx.As(new(ports.PutObjectService)),
		),

		fx.Annotate(
			use_cases.NewGetPublicObjectUseCase,
			fx.As(new(ports.GetPublicObjectService)),
		),

		fx.Annotate(
			use_cases.NewGetPrivateObjectUseCase,
			fx.As(new(ports.GetPrivateObjectService)),
		),

		fx.Annotate(
			use_cases.NewDeleteObjectUseCase,
			fx.As(new(ports.DeleteObjectService)),
		),
		fx.Annotate(
			use_cases.NewHeadObjectUseCase,
			fx.As(new(ports.HeadObjectService)),
		),
	),
)
