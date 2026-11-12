package presentation

import (
	"net/http"

	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/shared/config"
)

func provideProductHandler(
	create *use_cases.CreateProduct,
	get *use_cases.GetProductByID,
	update *use_cases.UpdateProduct,
	changeImage *use_cases.ChangeProductImage,
	changeVideo *use_cases.ChangeProductVideo,
	removeVideo *use_cases.RemoveProductVideo,
	deleteProduct *use_cases.DeleteProduct,
	list *use_cases.ListProducts,
	search *use_cases.SearchProducts,
	listStore *use_cases.ListProductsByStore,
	listMine *use_cases.ListMyProducts,
) *handlers.ProductHandler {
	return handlers.NewProductHandler(handlers.ProductUseCases{
		Create:      create,
		Get:         get,
		Update:      update,
		ChangeImage: changeImage,
		ChangeVideo: changeVideo,
		RemoveVideo: removeVideo,
		Delete:      deleteProduct,
		List:        list,
		Search:      search,
		ListStore:   listStore,
		ListMine:    listMine,
	})
}

// Module provides the Product HTTP handler and router.
var Module = fx.Module(
	"product-presentation",
	fx.Provide(
		provideProductHandler,
		fx.Annotate(
			NewRouter,
			fx.As(new(http.Handler)),
		),
	),
	fx.Invoke(validateHTTPConfiguration),
)

func validateHTTPConfiguration(cfg *config.Config) error {
	if cfg.AppPort == "" {
		return config.ErrMissingAppPort
	}
	return nil
}
