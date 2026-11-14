// Package use_cases contains application use cases for the Promotion service.
package use_cases

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/dtos"
	apperrors "github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/valueobjects"
)

const promotionCreatedEventType = "promotion.created"

// PromotionCreatedEvent contains the promotion reference sent to Product.
type PromotionCreatedEvent struct {
	PromoID   string  `json:"promo_id"`
	StoreID   string  `json:"store_id"`
	ProductID *string `json:"product_id"`
	Status    string  `json:"status"`
}

// CreatePromotionUseCase coordinates promotion creation and event recording.
type CreatePromotionUseCase struct {
	repository       ports.PromotionRepository
	outboxRepository ports.PromotionOutboxRepository
	unitOfWork       ports.UnitOfWork
}

// NewCreatePromotionUseCase constructs a promotion creation use case.
func NewCreatePromotionUseCase(
	repository ports.PromotionRepository,
	outboxRepository ports.PromotionOutboxRepository,
	unitOfWork ports.UnitOfWork,
) *CreatePromotionUseCase {
	return &CreatePromotionUseCase{
		repository:       repository,
		outboxRepository: outboxRepository,
		unitOfWork:       unitOfWork,
	}
}

// Execute creates a promotion and records its Product event atomically.
func (uc *CreatePromotionUseCase) Execute(
	ctx context.Context,
	input dtos.CreatePromotionInput,
) (dtos.PromotionDTO, error) {
	if uc == nil || uc.repository == nil {
		return dtos.PromotionDTO{}, apperrors.ErrPromotionRepositoryRequired
	}
	if uc.outboxRepository == nil {
		return dtos.PromotionDTO{}, apperrors.ErrPromotionOutboxRepositoryRequired
	}
	if uc.unitOfWork == nil {
		return dtos.PromotionDTO{}, apperrors.ErrPromotionUnitOfWorkRequired
	}

	var fixedAmount *valueobjects.Money

	if input.FixedAmount != nil {
		money, err := valueobjects.NewMoney(
			input.FixedAmount.Amount,
			input.FixedAmount.Currency,
		)
		if err != nil {
			return dtos.PromotionDTO{}, fmt.Errorf("create promotion money: %w", err)
		}
		fixedAmount = &money
	}

	discount, err := valueobjects.NewDiscount(
		input.PercentageBasisPoints,
		fixedAmount,
	)
	if err != nil {
		return dtos.PromotionDTO{}, fmt.Errorf("create promotion discount: %w", err)
	}

	promotion, err := entities.NewPromotion(
		input.StoreID,
		input.ProductID,
		input.CustomerID,
		discount,
		input.StartsAt,
		input.EndsAt,
	)
	if err != nil {
		return dtos.PromotionDTO{}, fmt.Errorf("create promotion entity: %w", err)
	}

	now := time.Now().UTC()

	event := PromotionCreatedEvent{
		PromoID:   promotion.ID(),
		StoreID:   promotion.StoreID(),
		ProductID: promotion.ProductID(),
		Status:    string(promotion.StatusAt(now)),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return dtos.PromotionDTO{}, fmt.Errorf("marshal promotion event: %w", err)
	}

	eventID := uuid.NewString()

	err = uc.unitOfWork.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.repository.CreateIfNoUnexpiredConflict(
			txCtx,
			promotion,
			now,
		); err != nil {
			return fmt.Errorf("persist promotion: %w", err)
		}

		if err := uc.outboxRepository.Create(
			txCtx,
			eventID,
			promotion.ID(),
			promotionCreatedEventType,
			payload,
		); err != nil {
			return fmt.Errorf("record promotion event: %w", err)
		}

		return nil
	})
	if err != nil {
		return dtos.PromotionDTO{}, fmt.Errorf("create promotion transaction: %w", err)
	}

	return toPromotionDTO(promotion, now), nil
}

// toPromotionDTO converts a domain promotion into its application DTO.
func toPromotionDTO(
	promotion *entities.Promotion,
	now time.Time,
) dtos.PromotionDTO {
	result := dtos.PromotionDTO{
		ID:         promotion.ID(),
		StoreID:    promotion.StoreID(),
		ProductID:  promotion.ProductID(),
		CustomerID: promotion.CustomerID(),
		StartsAt:   promotion.StartsAt(),
		EndsAt:     promotion.EndsAt(),
		Status:     promotion.StatusAt(now),
		CreatedAt:  promotion.CreatedAt(),
		UpdatedAt:  promotion.UpdatedAt(),
	}

	if percentage, ok := promotion.Discount().PercentageBasisPoints(); ok {
		result.PercentageBasisPoints = &percentage
	}

	if fixed, ok := promotion.Discount().FixedAmount(); ok {
		result.FixedAmount = &dtos.MoneyDTO{
			Amount:   fixed.Amount(),
			Currency: fixed.Currency(),
		}
	}

	return result
}
