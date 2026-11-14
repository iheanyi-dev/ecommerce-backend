package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/dtos"
	apperrors "github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/use_cases"
	presentation_errors "github.com/iheanyi-dev/ecommerce-backend/services/promotion/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/presentation/schemas"
)

type GetPromosForUserHandler struct {
	service interface {
		Execute(context.Context, dtos.GetPromosForUserInput) ([]dtos.PromotionDTO, error)
	}
}

func NewGetPromosForUserHandler(service *use_cases.GetPromosForUserUseCase) *GetPromosForUserHandler {
	return NewGetPromosForUserHandlerWithService(service)
}

func NewGetPromosForUserHandlerWithService(service interface {
	Execute(context.Context, dtos.GetPromosForUserInput) ([]dtos.PromotionDTO, error)
}) *GetPromosForUserHandler {
	return &GetPromosForUserHandler{service: service}
}

func (h *GetPromosForUserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		presentation_errors.WriteError(w, presentation_errors.ErrMethodNotAllowed)
		return
	}

	identity, ok := ports.AuthenticatedIdentityFromContext(r.Context())
	if !ok {
		presentation_errors.WriteError(w, presentation_errors.ErrAuthenticationRequired)
		return
	}

	limit, offset, err := parsePagination(r)
	if err != nil {
		presentation_errors.WriteError(w, err)
		return
	}

	result, err := h.service.Execute(r.Context(), dtos.GetPromosForUserInput{
		CustomerID: identity.UserID.String(),
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		presentation_errors.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, schemas.NewPromotionResponses(result))
}

func parsePagination(r *http.Request) (int, int, error) {
	limit, offset := 20, 0

	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			return 0, 0, apperrors.ErrInvalidPromotionQuery
		}
		limit = value
	}

	if raw := r.URL.Query().Get("offset"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			return 0, 0, apperrors.ErrInvalidPromotionQuery
		}
		offset = value
	}

	return limit, offset, nil
}
