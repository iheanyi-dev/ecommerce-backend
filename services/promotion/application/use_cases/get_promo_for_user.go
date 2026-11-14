package use_cases

import (
	"context"
	"fmt"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/dtos"
	apperrors "github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/entities"
)

// GetPromoForUserUseCase selects the highest-priority active promotion
// applicable to a customer buying a product from a store.
type GetPromoForUserUseCase struct {
	repository ports.PromotionRepository
}

func NewGetPromoForUserUseCase(repository ports.PromotionRepository) *GetPromoForUserUseCase {
	return &GetPromoForUserUseCase{repository: repository}
}

// Execute returns the best matching active promotion.
//
// Priority:
//  1. Store + product + customer.
//  2. Store + product.
//  3. Store + customer.
//  4. Store-wide.
func (uc *GetPromoForUserUseCase) Execute(
	ctx context.Context,
	input dtos.GetPromoForUserInput,
) (dtos.PromotionDTO, error) {
	if uc == nil || uc.repository == nil {
		return dtos.PromotionDTO{}, apperrors.ErrPromotionRepositoryRequired
	}

	for name, value := range map[string]string{
		"store ID":    input.StoreID,
		"product ID":  input.ProductID,
		"customer ID": input.CustomerID,
	} {
		if err := validateUUID(name, value); err != nil {
			return dtos.PromotionDTO{}, err
		}
	}

	now := time.Now().UTC()
	promotions, err := uc.repository.ListActiveForTarget(
		ctx,
		input.StoreID,
		input.ProductID,
		input.CustomerID,
		now,
	)
	if err != nil {
		return dtos.PromotionDTO{}, fmt.Errorf("get active promotions for user: %w", err)
	}

	var selected *entities.Promotion
	selectedPriority := 5

	for _, promotion := range promotions {
		if promotion == nil || !promotion.AppliesTo(
			input.StoreID,
			input.ProductID,
			input.CustomerID,
			now,
		) {
			continue
		}

		productID := promotion.ProductID()
		customerID := promotion.CustomerID()

		priority := 0
		switch {
		case productID != nil && customerID != nil:
			priority = 1
		case productID != nil:
			priority = 2
		case customerID != nil:
			priority = 3
		default:
			priority = 4
		}

		if priority < selectedPriority {
			selected = promotion
			selectedPriority = priority
		}
	}

	if selected == nil {
		return dtos.PromotionDTO{}, apperrors.ErrPromotionNotFound
	}

	return toPromotionDTO(selected, now), nil
}
