package use_cases

import (
	"context"
	"fmt"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/dtos"
	apperrors "github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/ports"
)

// GetPromosForUserUseCase retrieves promotions a customer can participate in
// across all stores, using pagination.
type GetPromosForUserUseCase struct {
	repository ports.PromotionRepository
}

func NewGetPromosForUserUseCase(repository ports.PromotionRepository) *GetPromosForUserUseCase {
	return &GetPromosForUserUseCase{repository: repository}
}

// Execute returns store-wide promotions and promotions explicitly targeting
// the customer. Product-specific negotiated promotions are included when
// their CustomerID matches. Product-only promotions are not included.
func (uc *GetPromosForUserUseCase) Execute(
	ctx context.Context,
	input dtos.GetPromosForUserInput,
) ([]dtos.PromotionDTO, error) {
	if uc == nil || uc.repository == nil {
		return nil, apperrors.ErrPromotionRepositoryRequired
	}
	if err := validateUUID("customer ID", input.CustomerID); err != nil {
		return nil, err
	}
	if err := validatePagination(input.Limit, input.Offset); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	promotions, err := uc.repository.ListAvailableForCustomer(
		ctx,
		input.CustomerID,
		now,
		input.Limit,
		input.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("get promotions for user: %w", err)
	}

	return toPromotionDTOs(promotions, now), nil
}
