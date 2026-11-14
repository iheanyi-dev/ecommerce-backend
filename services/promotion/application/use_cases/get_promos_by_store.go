package use_cases

import (
	"context"
	"fmt"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/dtos"
	apperrors "github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/ports"
)

// GetPromosByStoreUseCase retrieves a paginated list of a store's promotions.
type GetPromosByStoreUseCase struct {
	repository ports.PromotionRepository
}

func NewGetPromosByStoreUseCase(repository ports.PromotionRepository) *GetPromosByStoreUseCase {
	return &GetPromosByStoreUseCase{repository: repository}
}

// Execute returns all promotions belonging to the requested store,
// including scheduled, active, paused, expired, and cancelled promotions.
func (uc *GetPromosByStoreUseCase) Execute(
	ctx context.Context,
	input dtos.GetPromosByStoreInput,
) ([]dtos.PromotionDTO, error) {
	if uc == nil || uc.repository == nil {
		return nil, apperrors.ErrPromotionRepositoryRequired
	}
	if err := validateUUID("store ID", input.StoreID); err != nil {
		return nil, err
	}
	if err := validatePagination(input.Limit, input.Offset); err != nil {
		return nil, err
	}

	promotions, err := uc.repository.ListByStore(ctx, input.StoreID, input.Limit, input.Offset)
	if err != nil {
		return nil, fmt.Errorf("get promotions by store: %w", err)
	}

	return toPromotionDTOs(promotions, time.Now().UTC()), nil
}
