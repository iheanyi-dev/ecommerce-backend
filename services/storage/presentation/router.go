package presentation

import (
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/presentation/middleware"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/shared/config"
)

type Router struct {
	handler http.Handler
}

func NewRouter(
	getPublicObjectService ports.GetPublicObjectService,
	logger ports.Logger,
	metrics ports.Metrics,
	tracer ports.Tracer,
	cfg *config.Config,
) *Router {
	mux := http.NewServeMux()

	getPublicObjectHandler := handlers.NewGetPublicObjectHandler(
		getPublicObjectService,
	)

	// Only public-object retrieval is exposed through the HTTP/Gateway path.
	// The reference is a wildcard because storage references may contain
	// directory segments, for example "stores/<id>/image.png".
	mux.Handle(
		"GET /api/v1/storage/public/{reference...}",
		getPublicObjectHandler,
	)

	// Middleware order:
	//
	// RequestID
	//   -> ServiceAuthentication
	//      -> Tracing
	//         -> Observability
	//            -> Router
	//
	// Storage remains completely internal and therefore service
	// authentication protects the entire HTTP boundary.
	var handler http.Handler = mux

	handler = middleware.Observability(logger, metrics)(handler)
	handler = middleware.Tracing(tracer)(handler)
	handler = middleware.ServiceAuthentication(
		cfg.ServiceAuthName,
		cfg.ServiceAuthSecret,
	)(handler)
	handler = middleware.RequestID(handler)

	return &Router{
		handler: handler,
	}
}

func (r *Router) Handler() http.Handler {
	return r.handler
}
