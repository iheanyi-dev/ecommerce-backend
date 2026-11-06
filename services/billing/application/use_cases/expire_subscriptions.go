package use_cases

import (
	"context"
	"fmt"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
)

const defaultExpirationBatchSize int32 = 100

type ExpireSubscriptionsInput struct {
	Now       time.Time
	BatchSize int32
}

type ExpireSubscriptionsOutput struct {
	Expired int
}

// ExpireSubscriptions expires due subscriptions and records their Store events
// atomically. The repository locks selected rows to coordinate concurrent workers.
type ExpireSubscriptions struct {
	repository ports.SubscriptionRepository
	unitOfWork ports.UnitOfWork
	outbox     ports.SubscriptionOutboxRepository
	plans      ports.PlanRepository
}

func NewExpireSubscriptions(
	repository ports.SubscriptionRepository,
	unitOfWork ports.UnitOfWork,
	outbox ports.SubscriptionOutboxRepository,
	plans ports.PlanRepository,
) *ExpireSubscriptions {
	return &ExpireSubscriptions{
		repository: repository,
		unitOfWork: unitOfWork,
		outbox:     outbox,
		plans:      plans,
	}
}

func (useCase *ExpireSubscriptions) Execute(
	ctx context.Context,
	input ExpireSubscriptionsInput,
) (ExpireSubscriptionsOutput, error) {
	if useCase == nil || useCase.repository == nil {
		return ExpireSubscriptionsOutput{}, application.ErrSubscriptionRepositoryUnavailable
	}
	if useCase.unitOfWork == nil {
		return ExpireSubscriptionsOutput{}, fmt.Errorf("subscription unit of work is unavailable")
	}
	if useCase.outbox == nil {
		return ExpireSubscriptionsOutput{}, fmt.Errorf("subscription outbox repository is unavailable")
	}
	if useCase.plans == nil {
		return ExpireSubscriptionsOutput{}, fmt.Errorf("subscription plan repository is unavailable")
	}
	if input.Now.IsZero() {
		return ExpireSubscriptionsOutput{}, fmt.Errorf("expiration time is required")
	}
	if input.BatchSize == 0 {
		input.BatchSize = defaultExpirationBatchSize
	}
	if input.BatchSize < 0 {
		return ExpireSubscriptionsOutput{}, fmt.Errorf("expiration batch size must be positive")
	}

	now := input.Now.UTC()
	var output ExpireSubscriptionsOutput

	err := useCase.unitOfWork.WithinTransaction(ctx, func(txCtx context.Context) error {
		subscriptions, err := useCase.repository.GetDueForUpdate(txCtx, now, input.BatchSize)
		if err != nil {
			return fmt.Errorf("get due subscriptions: %w", err)
		}

		for _, subscription := range subscriptions {
			if err := subscription.Expire(now); err != nil {
				return fmt.Errorf("expire subscription %s: %w", subscription.ID, err)
			}
			if err := useCase.repository.Update(txCtx, subscription); err != nil {
				return fmt.Errorf("persist expired subscription %s: %w", subscription.ID, err)
			}
			if err := recordSubscriptionUpdate(txCtx, useCase.outbox, useCase.plans, subscription); err != nil {
				return fmt.Errorf("record expiration event for subscription %s: %w", subscription.ID, err)
			}
			output.Expired++
		}
		return nil
	})
	if err != nil {
		return ExpireSubscriptionsOutput{}, err
	}

	return output, nil
}
