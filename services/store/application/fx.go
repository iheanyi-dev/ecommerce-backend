package application

import (
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/use_cases"
)

// Module provides Store application-layer dependencies.
//
// Use cases depend only on application ports. Fx maps their concrete
// implementations to the service interfaces consumed by presentation.
var Module = fx.Module(
	"application",

	fx.Provide(
		fx.Annotate(
			use_cases.NewCreateStoreUseCase,
			fx.As(new(ports.CreateStoreService)),
		),

		fx.Annotate(
			use_cases.NewGetStoreByIDUseCase,
			fx.As(new(ports.GetStoreByIDService)),
		),

		fx.Annotate(
			use_cases.NewGetStoreByOwnerUseCase,
			fx.As(new(ports.GetStoreByOwnerService)),
		),

		fx.Annotate(
			use_cases.NewGetStoreBySlugUseCase,
			fx.As(new(ports.GetStoreBySlugService)),
		),

		fx.Annotate(
			use_cases.NewListStoresUseCase,
			fx.As(new(ports.ListStoresService)),
		),

		fx.Annotate(
			use_cases.NewUpdateStoreUseCase,
			fx.As(new(ports.UpdateStoreService)),
		),

		fx.Annotate(
			use_cases.NewChangeStoreImageUseCase,
			fx.As(new(ports.ChangeStoreImageService)),
		),

		fx.Annotate(
			use_cases.NewChangeStorePlanUseCase,
			fx.As(new(ports.ChangeStorePlanService)),
		),

		fx.Annotate(
			use_cases.NewChangeStoreStatusUseCase,
			fx.As(new(ports.ChangeStoreStatusService)),
		),

		fx.Annotate(
			use_cases.NewAddStoreKnowledgeUseCase,
			fx.As(new(ports.AddStoreKnowledgeService)),
		),

		fx.Annotate(
			use_cases.NewChatUseCase,
			fx.As(new(ports.ChatService)),
		),

		fx.Annotate(
			use_cases.NewRemoveStaleKnowledgeUseCase,
			fx.As(new(ports.RemoveStaleKnowledgeService)),
		),
	),
)
