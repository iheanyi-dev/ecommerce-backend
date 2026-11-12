package application

import (
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/use_cases"
)

// Module provides Product application use cases.
var Module = fx.Module(
	"product-application",
	fx.Provide(
		use_cases.NewCreateProduct,
		use_cases.NewGetProductByID,
		use_cases.NewUpdateProduct,
		use_cases.NewChangeProductImage,
		use_cases.NewChangeProductVideo,
		use_cases.NewRemoveProductVideo,
		use_cases.NewDeleteProduct,
		use_cases.NewListProducts,
		use_cases.NewSearchProducts,
		use_cases.NewSearchStoreProducts,
		use_cases.NewCountProductsByStore,
		use_cases.NewGetProductsByIDsForCart,
		use_cases.NewListProductsByStore,
		use_cases.NewListMyProducts,
	),
)
