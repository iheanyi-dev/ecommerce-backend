package use_cases

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
)

// CancelSubInput identifies the store subscription to cancel.
type CancelSubInput struct {
	StoreID        uuid.UUID
	SubscriptionID uuid.UUID
}

// CancelSub cancels a store subscription and records a Store event.
type CancelSub struct {
	repository ports.SubscriptionRepository
	unitOfWork ports.UnitOfWork
	outbox     ports.SubscriptionOutboxRepository
	plans      ports.PlanRepository
	telemetry  operationTelemetry
}

func NewCancelSub(
	repository ports.SubscriptionRepository,
	unitOfWork ports.UnitOfWork,
	outbox ports.SubscriptionOutboxRepository,
	plans ports.PlanRepository,
	observability ...ports.Observability,
) *CancelSub {
	var dependencies ports.Observability
	if len(observability) > 0 {
		dependencies = observability[0]
	}

	return &CancelSub{
		repository: repository,
		unitOfWork: unitOfWork,
		outbox:     outbox,
		plans:      plans,
		telemetry: newOperationTelemetry(
			dependencies.Logger,
			dependencies.Metrics,
			"cancel_sub",
			"billing_cancel_sub",
		),
	}
}

func (useCase *CancelSub) Execute(
	ctx context.Context,
	input CancelSubInput,
) (entities.Subscription, error) {
	if useCase == nil || useCase.repository == nil {
		return entities.Subscription{}, application.ErrSubscriptionRepositoryUnavailable
	}

	startedAt := useCase.telemetry.start(ctx)
	fail := func(category string, err error) (entities.Subscription, error) {
		useCase.telemetry.failure(ctx, startedAt, category)
		return entities.Subscription{}, err
	}

	if useCase.unitOfWork == nil {
		return fail("unit_of_work_unavailable", fmt.Errorf("subscription unit of work is unavailable"))
	}
	if useCase.plans == nil {
		return fail("plan_repository_unavailable", fmt.Errorf("subscription plan repository is unavailable"))
	}
	if useCase.outbox == nil {
		return fail("outbox_unavailable", fmt.Errorf("subscription outbox repository is unavailable"))
	}
	if input.StoreID == uuid.Nil || input.SubscriptionID == uuid.Nil {
		return fail("invalid_input", domain.ErrInvalidSubscription)
	}

	var result entities.Subscription
	err := useCase.unitOfWork.WithinTransaction(ctx, func(txCtx context.Context) error {
		subscription, err := useCase.repository.GetCurrentByStore(txCtx, input.StoreID)
		if err != nil {
			if errors.Is(err, application.ErrSubscriptionNotFound) {
				return application.ErrSubscriptionNotFound
			}
			return fmt.Errorf("get current subscription: %w", err)
		}
		if subscription == nil ||
			subscription.StoreID != input.StoreID ||
			subscription.ID != input.SubscriptionID {
			return application.ErrSubscriptionNotFound
		}

		if err := subscription.Cancel(timeNowUTC()); err != nil {
			return err
		}
		if err := useCase.repository.Update(txCtx, *subscription); err != nil {
			return fmt.Errorf("persist subscription cancellation: %w", err)
		}
		if err := recordSubscriptionUpdate(txCtx, useCase.outbox, useCase.plans, *subscription); err != nil {
			return err
		}

		result = *subscription
		return nil
	})
	if err != nil {
		return fail("persistence", err)
	}

	useCase.telemetry.success(ctx, startedAt)
	return result, nil
}
