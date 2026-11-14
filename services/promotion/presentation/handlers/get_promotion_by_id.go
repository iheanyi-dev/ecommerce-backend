package handlers

import (
	"context"
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/dtos"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/use_cases"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/promotion/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/presentation/schemas"
)

type GetPromotionByIDHandler struct {
	service interface {
		Execute(context.Context, dtos.GetPromotionByIDInput) (dtos.PromotionDTO, error)
	}
}

func NewGetPromotionByIDHandler(service *use_cases.GetPromotionByIDUseCase) *GetPromotionByIDHandler {
	return NewGetPromotionByIDHandlerWithService(service)
}

func NewGetPromotionByIDHandlerWithService(service interface {
	Execute(context.Context, dtos.GetPromotionByIDInput) (dtos.PromotionDTO, error)
}) *GetPromotionByIDHandler {
	return &GetPromotionByIDHandler{service: service}
}

func (h *GetPromotionByIDHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		presentation_errors.WriteError(w, presentation_errors.ErrMethodNotAllowed)
		return
	}

	result, err := h.service.Execute(r.Context(), dtos.GetPromotionByIDInput{
		PromotionID: r.PathValue("id"),
	})
	if err != nil {
		presentation_errors.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, schemas.NewPromotionResponse(result))
}
