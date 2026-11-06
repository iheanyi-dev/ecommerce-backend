package use_cases

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
)

const (
	subscriptionStatusPending   = "pending"
	subscriptionStatusActive    = "active"
	subscriptionStatusFailed    = "failed"
	subscriptionStatusExpired   = "expired"
	subscriptionStatusCancelled = "cancelled"
)

// ErrInvalidSubscriptionStatus indicates that a subscription update contains
// a status that Store does not recognize.
var ErrInvalidSubscriptionStatus = fmt.Errorf("invalid subscription status")

// UpdateAfterSubscriptionUseCase applies trusted Billing subscription
// updates to Store aggregates.
//
// Pending and failed subscriptions do not change Store state. Active
// subscriptions apply their plan and activate the Store. Expired and
// cancelled subscriptions deactivate the Store without changing its plan.
type UpdateAfterSubscriptionUseCase struct {
	storeRepository ports.StoreRepository
	logger          ports.Logger
	metrics         ports.Metrics
}

// NewUpdateAfterSubscriptionUseCase constructs the subscription update
// use case.
func NewUpdateAfterSubscriptionUseCase(
	storeRepository ports.StoreRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *UpdateAfterSubscriptionUseCase {
	return &UpdateAfterSubscriptionUseCase{
		storeRepository: storeRepository,
		logger:          logger,
		metrics:         metrics,
	}
}

// Execute applies a subscription status update to the associated Store.
func (uc *UpdateAfterSubscriptionUseCase) Execute(
	ctx context.Context,
	input dto.UpdateAfterSubscriptionInput,
) (dto.StoreOutput, error) {
	startedAt := time.Now()

	uc.increment(ctx, ports.Metric{
		Name:  "store_update_after_subscription_total",
		Value: 1,
	})

	if input.EventID == uuid.Nil ||
		input.SubscriptionID == uuid.Nil ||
		input.StoreID == uuid.Nil {
		return uc.fail(
			ctx,
			startedAt,
			"invalid_event",
			fmt.Errorf("event, subscription, and store IDs are required"),
		)
	}

	switch input.Status {
	case subscriptionStatusPending, subscriptionStatusFailed:
		// These states do not affect an existing Store subscription.
		output, err := uc.findStore(ctx, input.StoreID)
		if err != nil {
			return uc.fail(ctx, startedAt, "store_lookup", err)
		}

		uc.succeed(ctx, startedAt, input)
		return output, nil

	case subscriptionStatusActive:
		// Continue below.

	case subscriptionStatusExpired, subscriptionStatusCancelled:
		// Continue below.

	default:
		return uc.fail(
			ctx,
			startedAt,
			"invalid_subscription_status",
			ErrInvalidSubscriptionStatus,
		)
	}

	store, err := uc.storeRepository.FindByID(ctx, input.StoreID)
	if err != nil {
		return uc.fail(
			ctx,
			startedAt,
			"store_lookup",
			fmt.Errorf("find store: %w", err),
		)
	}

	if store == nil {
		return uc.fail(
			ctx,
			startedAt,
			"store_not_found",
			applicationerrors.ErrStoreNotFound,
		)
	}

	switch input.Status {
	case subscriptionStatusActive:
		targetPlan, err := valueobjects.NewPlan(
			valueobjects.PlanType(input.PlanType),
		)
		if err != nil {
			return uc.fail(ctx, startedAt, "invalid_plan", err)
		}

		if err := store.ChangePlan(targetPlan); err != nil {
			return uc.fail(ctx, startedAt, "plan_change", err)
		}

		if err := store.Activate(); err != nil {
			return uc.fail(ctx, startedAt, "status_transition", err)
		}

	case subscriptionStatusExpired, subscriptionStatusCancelled:
		if err := store.Deactivate(); err != nil {
			return uc.fail(ctx, startedAt, "status_transition", err)
		}
	}

	// ChangePlan persists the complete aggregate, including its status.
	if err := uc.storeRepository.ChangePlan(ctx, store); err != nil {
		return uc.fail(
			ctx,
			startedAt,
			"store_persistence",
			fmt.Errorf("persist subscription update: %w", err),
		)
	}

	output := toStoreOutput(store)

	uc.succeed(ctx, startedAt, input)
	return output, nil
}

func (uc *UpdateAfterSubscriptionUseCase) findStore(
	ctx context.Context,
	storeID uuid.UUID,
) (dto.StoreOutput, error) {
	store, err := uc.storeRepository.FindByID(ctx, storeID)
	if err != nil {
		return dto.StoreOutput{}, fmt.Errorf("find store: %w", err)
	}

	if store == nil {
		return dto.StoreOutput{}, applicationerrors.ErrStoreNotFound
	}

	return toStoreOutput(store), nil
}

func (uc *UpdateAfterSubscriptionUseCase) succeed(
	ctx context.Context,
	startedAt time.Time,
	input dto.UpdateAfterSubscriptionInput,
) {
	uc.log(ctx, ports.LogEvent{
		Event:     "store.update_after_subscription.succeeded",
		Operation: "update_after_subscription",
	})

	uc.increment(ctx, ports.Metric{
		Name:  "store_update_after_subscription_success_total",
		Value: 1,
	})

	uc.observe(ctx, ports.Metric{
		Name:  "store_update_after_subscription_duration_seconds",
		Value: time.Since(startedAt).Seconds(),
	})
}

func (uc *UpdateAfterSubscriptionUseCase) fail(
	ctx context.Context,
	startedAt time.Time,
	category string,
	err error,
) (dto.StoreOutput, error) {
	uc.log(ctx, ports.LogEvent{
		Event:           "store.update_after_subscription.failed",
		Operation:       "update_after_subscription",
		FailureCategory: category,
	})

	uc.increment(ctx, ports.Metric{
		Name:  "store_update_after_subscription_failure_total",
		Value: 1,
		Labels: map[string]string{
			"failure_category": category,
		},
	})

	uc.observe(ctx, ports.Metric{
		Name:  "store_update_after_subscription_duration_seconds",
		Value: time.Since(startedAt).Seconds(),
	})

	return dto.StoreOutput{}, err
}

func (uc *UpdateAfterSubscriptionUseCase) log(
	ctx context.Context,
	event ports.LogEvent,
) {
	if uc.logger != nil {
		_ = uc.logger.Log(ctx, event)
	}
}

func (uc *UpdateAfterSubscriptionUseCase) increment(
	ctx context.Context,
	metric ports.Metric,
) {
	if uc.metrics != nil {
		_ = uc.metrics.Increment(ctx, metric)
	}
}

func (uc *UpdateAfterSubscriptionUseCase) observe(
	ctx context.Context,
	metric ports.Metric,
) {
	if uc.metrics != nil {
		_ = uc.metrics.Observe(ctx, metric)
	}
}

var _ ports.UpdateAfterSubscriptionService = (*UpdateAfterSubscriptionUseCase)(nil)
