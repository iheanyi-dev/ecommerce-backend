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
)

const (
	promotionCancelledEventType = "promotion.cancelled"
	promotionActivatedEventType = "promotion.activated"
	promotionExpiredEventType   = "promotion.expired"
	promotionRenewedEventType   = "promotion.renewed"
)

// PromotionLifecycleEvent is the Product-facing promotion snapshot reference.
type PromotionLifecycleEvent struct {
	PromoID   string  `json:"promo_id"`
	StoreID   string  `json:"store_id"`
	ProductID *string `json:"product_id"`
	Status    string  `json:"status"`
}

type promotionTransition func(*entities.Promotion, time.Time) error

type promotionLifecycle struct {
	repository       ports.PromotionRepository
	outboxRepository ports.PromotionOutboxRepository
	unitOfWork       ports.UnitOfWork
}

func (uc *promotionLifecycle) execute(
	ctx context.Context,
	promotionID string,
	eventType string,
	transition promotionTransition,
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
	if err := validateUUID("promotion ID", promotionID); err != nil {
		return dtos.PromotionDTO{}, err
	}

	now := time.Now().UTC()
	eventID := uuid.NewString()
	var updated *entities.Promotion

	err := uc.unitOfWork.WithinTransaction(ctx, func(txCtx context.Context) error {
		promotion, err := uc.repository.GetByID(txCtx, promotionID)
		if err != nil {
			return fmt.Errorf("get promotion: %w", err)
		}
		if promotion == nil {
			return apperrors.ErrPromotionNotFound
		}

		if err := transition(promotion, now); err != nil {
			return fmt.Errorf("transition promotion: %w", err)
		}

		if err := uc.repository.Update(txCtx, promotion); err != nil {
			return fmt.Errorf("update promotion: %w", err)
		}

		payload, err := json.Marshal(PromotionLifecycleEvent{
			PromoID:   promotion.ID(),
			StoreID:   promotion.StoreID(),
			ProductID: promotion.ProductID(),
			Status:    string(promotion.StatusAt(now)),
		})
		if err != nil {
			return fmt.Errorf("marshal promotion event: %w", err)
		}

		if err := uc.outboxRepository.Create(
			txCtx,
			eventID,
			promotion.ID(),
			eventType,
			payload,
		); err != nil {
			return fmt.Errorf("record promotion event: %w", err)
		}

		updated = promotion
		return nil
	})
	if err != nil {
		return dtos.PromotionDTO{}, fmt.Errorf("execute promotion lifecycle: %w", err)
	}

	return toPromotionDTO(updated, now), nil
}

// CancelPromotionUseCase cancels a promotion and records its Product event.
type CancelPromotionUseCase struct {
	lifecycle *promotionLifecycle
}

func NewCancelPromotionUseCase(
	repository ports.PromotionRepository,
	outbox ports.PromotionOutboxRepository,
	unitOfWork ports.UnitOfWork,
) *CancelPromotionUseCase {
	return &CancelPromotionUseCase{lifecycle: &promotionLifecycle{
		repository: repository, outboxRepository: outbox, unitOfWork: unitOfWork,
	}}
}

func (uc *CancelPromotionUseCase) Execute(
	ctx context.Context, input dtos.CancelPromotionInput,
) (dtos.PromotionDTO, error) {
	if uc == nil {
		return dtos.PromotionDTO{}, apperrors.ErrPromotionRepositoryRequired
	}
	return uc.lifecycle.execute(ctx, input.PromotionID, promotionCancelledEventType,
		func(p *entities.Promotion, now time.Time) error { return p.Cancel(now) })
}

// ActivatePromotionUseCase activates a scheduled promotion and records its Product event.
type ActivatePromotionUseCase struct {
	lifecycle *promotionLifecycle
}

func NewActivatePromotionUseCase(
	repository ports.PromotionRepository,
	outbox ports.PromotionOutboxRepository,
	unitOfWork ports.UnitOfWork,
) *ActivatePromotionUseCase {
	return &ActivatePromotionUseCase{lifecycle: &promotionLifecycle{
		repository: repository, outboxRepository: outbox, unitOfWork: unitOfWork,
	}}
}

func (uc *ActivatePromotionUseCase) Execute(
	ctx context.Context, input dtos.ActivatePromotionInput,
) (dtos.PromotionDTO, error) {
	if uc == nil {
		return dtos.PromotionDTO{}, apperrors.ErrPromotionRepositoryRequired
	}
	return uc.lifecycle.execute(ctx, input.PromotionID, promotionActivatedEventType,
		func(p *entities.Promotion, now time.Time) error { return p.Activate(now) })
}

// ExpirePromotionUseCase expires a promotion and records its Product removal event.
type ExpirePromotionUseCase struct {
	lifecycle *promotionLifecycle
}

func NewExpirePromotionUseCase(
	repository ports.PromotionRepository,
	outbox ports.PromotionOutboxRepository,
	unitOfWork ports.UnitOfWork,
) *ExpirePromotionUseCase {
	return &ExpirePromotionUseCase{lifecycle: &promotionLifecycle{
		repository: repository, outboxRepository: outbox, unitOfWork: unitOfWork,
	}}
}

func (uc *ExpirePromotionUseCase) Execute(
	ctx context.Context, input dtos.ExpirePromotionInput,
) (dtos.PromotionDTO, error) {
	if uc == nil {
		return dtos.PromotionDTO{}, apperrors.ErrPromotionRepositoryRequired
	}
	return uc.lifecycle.execute(ctx, input.PromotionID, promotionExpiredEventType,
		func(p *entities.Promotion, now time.Time) error { return p.Expire(now) })
}

// RenewPromotionUseCase renews an expired promotion without changing its ID.
type RenewPromotionUseCase struct {
	lifecycle *promotionLifecycle
}

func NewRenewPromotionUseCase(
	repository ports.PromotionRepository,
	outbox ports.PromotionOutboxRepository,
	unitOfWork ports.UnitOfWork,
) *RenewPromotionUseCase {
	return &RenewPromotionUseCase{lifecycle: &promotionLifecycle{
		repository: repository, outboxRepository: outbox, unitOfWork: unitOfWork,
	}}
}

func (uc *RenewPromotionUseCase) Execute(
	ctx context.Context, input dtos.RenewPromotionInput,
) (dtos.PromotionDTO, error) {
	if uc == nil {
		return dtos.PromotionDTO{}, apperrors.ErrPromotionRepositoryRequired
	}
	return uc.lifecycle.execute(ctx, input.PromotionID, promotionRenewedEventType,
		func(p *entities.Promotion, now time.Time) error {
			return p.Renew(input.StartsAt, input.EndsAt, now)
		})
}

// DeletePromotionUseCase permanently deletes an expired or cancelled promotion.
type DeletePromotionUseCase struct {
	repository ports.PromotionRepository
	unitOfWork ports.UnitOfWork
}

func NewDeletePromotionUseCase(
	repository ports.PromotionRepository,
	unitOfWork ports.UnitOfWork,
) *DeletePromotionUseCase {
	return &DeletePromotionUseCase{repository: repository, unitOfWork: unitOfWork}
}

// Execute deletes an expired or cancelled promotion without publishing an event.
// Product snapshots are removed by the earlier cancellation or expiration event.
func (uc *DeletePromotionUseCase) Execute(
	ctx context.Context, input dtos.DeletePromotionInput,
) error {
	if uc == nil || uc.repository == nil {
		return apperrors.ErrPromotionRepositoryRequired
	}
	if uc.unitOfWork == nil {
		return apperrors.ErrPromotionUnitOfWorkRequired
	}
	if err := validateUUID("promotion ID", input.PromotionID); err != nil {
		return err
	}

	now := time.Now().UTC()
	err := uc.unitOfWork.WithinTransaction(ctx, func(txCtx context.Context) error {
		promotion, err := uc.repository.GetByID(txCtx, input.PromotionID)
		if err != nil {
			return fmt.Errorf("get promotion: %w", err)
		}
		if promotion == nil {
			return apperrors.ErrPromotionNotFound
		}

		status := promotion.StatusAt(now)
		if status != entities.PromotionExpired && status != entities.PromotionCancelled {
			return apperrors.ErrPromotionNotDeletable
		}

		if err := uc.repository.Delete(txCtx, input.PromotionID); err != nil {
			return fmt.Errorf("delete promotion: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("delete promotion transaction: %w", err)
	}
	return nil
}
