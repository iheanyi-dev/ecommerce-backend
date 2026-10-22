package presentation

import (
	"net/http"

	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/presentation/middleware"
)

// Module wires Cart presentation handlers and routing.
var Module = fx.Module(
	"cart-presentation",
	fx.Provide(
		use_cases.NewGetMyCart,
		use_cases.NewAddItemToCart,
		use_cases.NewSetCartItemQuantity,
		use_cases.NewRemoveCartItem,
		use_cases.NewClearCart,
		use_cases.NewCheckoutCart,

		fx.Annotate(
			handlers.NewGetMyCartHandler,
			fx.As(new(http.Handler)),
			fx.ResultTags(`name:"getMyCartHandler"`),
		),
		fx.Annotate(
			handlers.NewAddItemToCartHandler,
			fx.As(new(http.Handler)),
			fx.ResultTags(`name:"addItemToCartHandler"`),
		),
		fx.Annotate(
			handlers.NewSetCartItemQuantityHandler,
			fx.As(new(http.Handler)),
			fx.ResultTags(`name:"setCartItemQuantityHandler"`),
		),
		fx.Annotate(
			handlers.NewRemoveCartItemHandler,
			fx.As(new(http.Handler)),
			fx.ResultTags(`name:"removeCartItemHandler"`),
		),
		fx.Annotate(
			handlers.NewClearCartHandler,
			fx.As(new(http.Handler)),
			fx.ResultTags(`name:"clearCartHandler"`),
		),
		fx.Annotate(
			handlers.NewCheckoutCartHandler,
			fx.As(new(http.Handler)),
			fx.ResultTags(`name:"checkoutCartHandler"`),
		),

		middleware.NewServiceAuthenticationMiddleware,
		middleware.NewIdentityPropagationMiddleware,
		middleware.NewRequestIDMiddleware,
		middleware.NewTracingMiddleware,

		fx.Annotate(
			NewRouter,
			fx.ParamTags(
				`name:"getMyCartHandler"`,
				`name:"addItemToCartHandler"`,
				`name:"setCartItemQuantityHandler"`,
				`name:"removeCartItemHandler"`,
				`name:"clearCartHandler"`,
				`name:"checkoutCartHandler"`,
				"",
				"",
				"",
				"",
			),
		),
	),
	fx.Invoke(RegisterHTTPServerLifecycle),
)
