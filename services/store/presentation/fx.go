package presentation

import (
	"net/http"

	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/middleware"
)

// Module provides Store presentation-layer dependencies.
//
// Each handler is exposed as a named http.Handler because NewRouter accepts
// interface-based handlers. Naming prevents Fx from treating the twelve
// handlers as competing providers of the same http.Handler type.
var Module = fx.Module(
	"presentation",

	fx.Provide(
		fx.Annotate(
			handlers.NewCreateStoreHandler,
			fx.As(new(http.Handler)),
			fx.ResultTags(`name:"createStoreHandler"`),
		),
		fx.Annotate(
			handlers.NewGetStoreByIDHandler,
			fx.As(new(http.Handler)),
			fx.ResultTags(`name:"getStoreByIDHandler"`),
		),
		fx.Annotate(
			handlers.NewGetStoreByOwnerHandler,
			fx.As(new(http.Handler)),
			fx.ResultTags(`name:"getStoreByOwnerHandler"`),
		),
		fx.Annotate(
			handlers.NewGetStoreBySlugHandler,
			fx.As(new(http.Handler)),
			fx.ResultTags(`name:"getStoreBySlugHandler"`),
		),
		fx.Annotate(
			handlers.NewListStoresHandler,
			fx.As(new(http.Handler)),
			fx.ResultTags(`name:"listStoresHandler"`),
		),
		fx.Annotate(
			handlers.NewUpdateStoreHandler,
			fx.As(new(http.Handler)),
			fx.ResultTags(`name:"updateStoreHandler"`),
		),
		fx.Annotate(
			handlers.NewChangeStoreImageHandler,
			fx.As(new(http.Handler)),
			fx.ResultTags(`name:"changeStoreImageHandler"`),
		),
		fx.Annotate(
			handlers.NewChangeStorePlanHandler,
			fx.As(new(http.Handler)),
			fx.ResultTags(`name:"changeStorePlanHandler"`),
		),
		fx.Annotate(
			handlers.NewChangeStoreStatusHandler,
			fx.As(new(http.Handler)),
			fx.ResultTags(`name:"changeStoreStatusHandler"`),
		),
		fx.Annotate(
			handlers.NewAddStoreKnowledgeHandler,
			fx.As(new(http.Handler)),
			fx.ResultTags(`name:"addStoreKnowledgeHandler"`),
		),
		fx.Annotate(
			handlers.NewChatHandler,
			fx.As(new(http.Handler)),
			fx.ResultTags(`name:"chatHandler"`),
		),
		fx.Annotate(
			handlers.NewRemoveStaleKnowledgeHandler,
			fx.As(new(http.Handler)),
			fx.ResultTags(`name:"removeStaleKnowledgeHandler"`),
		),

		// Request ID establishes correlation for every request, including
		// requests rejected by the Store service authentication boundary.
		middleware.NewRequestIDMiddleware,

		// Tracing extracts the incoming trace context and creates the
		// transport-level Store request span.
		middleware.NewTracingMiddleware,

		// HTTP observability records transport-level request status,
		// duration, metrics, and safe request identity information.
		middleware.NewRequestObservabilityMiddleware,

		// Service authentication protects the complete Store service
		// boundary from unauthenticated external callers.
		middleware.NewServiceAuthenticationMiddleware,

		fx.Annotate(
			NewRouter,
			fx.ParamTags(
				// Twelve named HTTP handlers.
				`name:"createStoreHandler"`,
				`name:"getStoreByIDHandler"`,
				`name:"getStoreByOwnerHandler"`,
				`name:"getStoreBySlugHandler"`,
				`name:"listStoresHandler"`,
				`name:"updateStoreHandler"`,
				`name:"changeStoreImageHandler"`,
				`name:"changeStorePlanHandler"`,
				`name:"changeStoreStatusHandler"`,
				`name:"addStoreKnowledgeHandler"`,
				`name:"chatHandler"`,
				`name:"removeStaleKnowledgeHandler"`,

				// Four concrete middleware dependencies.
				//
				// These are intentionally untagged because they are provided
				// by their concrete types rather than as named http.Handler
				// values.
				"",
				"",
				"",
				"",
			),
		),
	),
)
