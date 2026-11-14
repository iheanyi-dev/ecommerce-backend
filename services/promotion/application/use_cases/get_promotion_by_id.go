// Package use_cases contains application use cases for the Promotion service.
package use_cases

import (
	"context"
	"fmt"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/dtos"
	apperrors "github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/ports"
)

// GetPromotionByIDUseCase retrieves a promotion by its unique ID.
type GetPromotionByIDUseCase struct {
	repository ports.PromotionRepository
}

// NewGetPromotionByIDUseCase constructs a promotion retrieval use case.
func NewGetPromotionByIDUseCase(
	repository ports.PromotionRepository,
) *GetPromotionByIDUseCase {
	return &GetPromotionByIDUseCase{repository: repository}
}

// Execute retrieves a promotion and converts it to its application DTO.
func (uc *GetPromotionByIDUseCase) Execute(
	ctx context.Context,
	input dtos.GetPromotionByIDInput,
) (dtos.PromotionDTO, error) {
	if uc == nil || uc.repository == nil {
		return dtos.PromotionDTO{}, apperrors.ErrPromotionRepositoryRequired
	}

	if err := validateUUID("promotion ID", input.PromotionID); err != nil {
		return dtos.PromotionDTO{}, err
	}

	promotion, err := uc.repository.GetByID(ctx, input.PromotionID)
	if err != nil {
		return dtos.PromotionDTO{}, fmt.Errorf("get promotion by ID: %w", err)
	}
	if promotion == nil {
		return dtos.PromotionDTO{}, apperrors.ErrPromotionNotFound
	}

	return toPromotionDTO(promotion, time.Now().UTC()), nil
}
