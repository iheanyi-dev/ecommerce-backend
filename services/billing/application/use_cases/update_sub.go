package use_cases

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
)

type UpdateSubInput struct {
	BusinessReferenceID string
	Status              entities.SubscriptionStatus
}

type UpdateSubOutput struct {
	Subscription entities.Subscription
}

// UpdateSub applies a verified payment outcome and records a Store event.
type UpdateSub struct {
	repository ports.SubscriptionRepository
	unitOfWork ports.UnitOfWork
	outbox     ports.SubscriptionOutboxRepository
	plans      ports.PlanRepository
}

func NewUpdateSub(
	repository ports.SubscriptionRepository,
	unitOfWork ports.UnitOfWork,
	outbox ports.SubscriptionOutboxRepository,
	plans ports.PlanRepository,
) *UpdateSub {
	return &UpdateSub{
		repository: repository,
		unitOfWork: unitOfWork,
		outbox:     outbox,
		plans:      plans,
	}
}

func (useCase *UpdateSub) Execute(
	ctx context.Context,
	input UpdateSubInput,
) (UpdateSubOutput, error) {
	if useCase == nil || useCase.repository == nil {
		return UpdateSubOutput{}, application.ErrSubscriptionRepositoryUnavailable
	}
	if useCase.unitOfWork == nil {
		return UpdateSubOutput{}, fmt.Errorf("subscription unit of work is unavailable")
	}
	if useCase.plans == nil {
		return UpdateSubOutput{}, fmt.Errorf("subscription plan repository is unavailable")
	}
	if useCase.outbox == nil {
		return UpdateSubOutput{}, fmt.Errorf("subscription outbox repository is unavailable")
	}

	subscriptionID, err := uuid.Parse(input.BusinessReferenceID)
	if err != nil || subscriptionID == uuid.Nil {
		return UpdateSubOutput{}, domain.ErrInvalidSubscription
	}
	if input.Status != entities.SubscriptionStatusActive &&
		input.Status != entities.SubscriptionStatusFailed {
		return UpdateSubOutput{}, domain.ErrInvalidSubscriptionStatus
	}

	var result UpdateSubOutput
	err = useCase.unitOfWork.WithinTransaction(ctx, func(txCtx context.Context) error {
		subscription, err := useCase.repository.GetByID(txCtx, subscriptionID)
		if err != nil {
			if errors.Is(err, application.ErrSubscriptionNotFound) {
				return application.ErrSubscriptionNotFound
			}
			return fmt.Errorf("get subscription: %w", err)
		}
		if subscription == nil || subscription.ID != subscriptionID {
			return application.ErrSubscriptionNotFound
		}

		now := time.Now().UTC()
		changed := false

		switch input.Status {
		case entities.SubscriptionStatusActive:
			switch subscription.Status {
			case entities.SubscriptionStatusPending:
				err = subscription.Activate(now)
				changed = err == nil
			case entities.SubscriptionStatusActive, entities.SubscriptionStatusExpired:
				// A renewal is a new subscription with a new business reference.
				// A late or duplicate event must not extend this subscription.
			default:
				return domain.ErrInvalidSubscriptionStatus
			}
		case entities.SubscriptionStatusFailed:
			switch subscription.Status {
			case entities.SubscriptionStatusPending:
				err = subscription.MarkPaymentFailed(now)
				changed = err == nil
			case entities.SubscriptionStatusActive,
				entities.SubscriptionStatusExpired,
				entities.SubscriptionStatusFailed:
				// A failed payment must not alter an existing paid subscription.
			default:
				return domain.ErrInvalidSubscriptionStatus
			}
		}

		if err != nil {
			return fmt.Errorf("apply subscription payment outcome: %w", err)
		}
		if changed {
			if err := useCase.repository.Update(txCtx, *subscription); err != nil {
				return fmt.Errorf("persist subscription update: %w", err)
			}
			if err := recordSubscriptionUpdate(txCtx, useCase.outbox, useCase.plans, *subscription); err != nil {
				return err
			}
		}

		result.Subscription = *subscription
		return nil
	})
	if err != nil {
		return UpdateSubOutput{}, err
	}
	return result, nil
}
