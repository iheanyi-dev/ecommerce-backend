package handlers

import (
	"context"
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/dtos"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/use_cases"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/promotion/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/presentation/schemas"
)

type GetPromoForUserHandler struct {
	service interface {
		Execute(context.Context, dtos.GetPromoForUserInput) (dtos.PromotionDTO, error)
	}
}

func NewGetPromoForUserHandler(service *use_cases.GetPromoForUserUseCase) *GetPromoForUserHandler {
	return NewGetPromoForUserHandlerWithService(service)
}

func NewGetPromoForUserHandlerWithService(service interface {
	Execute(context.Context, dtos.GetPromoForUserInput) (dtos.PromotionDTO, error)
}) *GetPromoForUserHandler {
	return &GetPromoForUserHandler{service: service}
}

func (h *GetPromoForUserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		presentation_errors.WriteError(w, presentation_errors.ErrMethodNotAllowed)
		return
	}

	identity, ok := ports.AuthenticatedIdentityFromContext(r.Context())
	if !ok {
		presentation_errors.WriteError(w, presentation_errors.ErrAuthenticationRequired)
		return
	}

	query := r.URL.Query()
	result, err := h.service.Execute(r.Context(), dtos.GetPromoForUserInput{
		StoreID:    query.Get("store_id"),
		ProductID:  query.Get("product_id"),
		CustomerID: identity.UserID.String(),
	})
	if err != nil {
		presentation_errors.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, schemas.NewPromotionResponse(result))
}
