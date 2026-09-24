package use_cases

import (
	"context"
	"fmt"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
)

// ChangeStoreStatusUseCase coordinates a trusted internal Store status change.
//
// The caller is expected to have already been authenticated and authorized at
// the presentation/service boundary. In the current architecture, Billing or
// Subscription determines when a Store must become active or inactive.
//
// This use case is therefore deliberately unaware of:
//   - authenticated user identity
//   - user roles
//   - subscription state
//   - payment state
//   - Gateway authentication
//   - HTTP or gRPC transport
//
// Its responsibility is to load the Store aggregate, validate and apply the
// requested domain status transition, persist the resulting status, and emit
// application observability.
type ChangeStoreStatusUseCase struct {
	storeRepository ports.StoreRepository
	logger          ports.Logger
	metrics         ports.Metrics
}

// NewChangeStoreStatusUseCase constructs the Change Store Status use case.
func NewChangeStoreStatusUseCase(
	storeRepository ports.StoreRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *ChangeStoreStatusUseCase {
	return &ChangeStoreStatusUseCase{
		storeRepository: storeRepository,
		logger:          logger,
		metrics:         metrics,
	}
}

// Execute changes the status of a Store.
//
// The status is supplied as a string at the application boundary and is
// converted into the domain Status value object before any aggregate mutation
// occurs.
func (uc *ChangeStoreStatusUseCase) Execute(
	ctx context.Context,
	input dto.ChangeStoreStatusInput,
) (dto.StoreOutput, error) {
	startedAt := time.Now()

	uc.increment(ctx, ports.Metric{
		Name:  "store_change_status_total",
		Value: 1,
	})

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

	targetStatus, err := valueobjects.NewStatus(
		valueobjects.Status(input.Status),
	)
	if err != nil {
		return uc.fail(
			ctx,
			startedAt,
			"invalid_status",
			err,
		)
	}

	switch targetStatus {
	case valueobjects.StatusActive:
		if err := store.Activate(); err != nil {
			return uc.fail(
				ctx,
				startedAt,
				"status_transition",
				err,
			)
		}

	case valueobjects.StatusInactive:
		if err := store.Deactivate(); err != nil {
			return uc.fail(
				ctx,
				startedAt,
				"status_transition",
				err,
			)
		}

	default:
		// NewStatus currently guarantees that only supported statuses reach
		// this point. Keep the default defensive so future domain statuses
		// cannot silently bypass the persistence boundary.
		return uc.fail(
			ctx,
			startedAt,
			"invalid_status",
			fmt.Errorf("unsupported store status: %s", targetStatus),
		)
	}

	if err := uc.storeRepository.ChangeStatus(
		ctx,
		store.ID().Value(),
		string(store.Status()),
	); err != nil {
		return uc.fail(
			ctx,
			startedAt,
			"store_persistence",
			fmt.Errorf("change store status: %w", err),
		)
	}

	output := toStoreOutput(store)

	uc.log(ctx, ports.LogEvent{
		Event:     "store.change_status.succeeded",
		Operation: "change_store_status",
	})

	uc.increment(ctx, ports.Metric{
		Name:  "store_change_status_success_total",
		Value: 1,
	})

	uc.observe(ctx, ports.Metric{
		Name:  "store_change_status_duration_seconds",
		Value: time.Since(startedAt).Seconds(),
	})

	return output, nil
}

// fail records bounded failure observability and returns the original
// application/domain error to the caller.
//
// No user identity is extracted here because Change Store Status is an
// internal trusted workflow rather than a user-authenticated Store operation.
func (uc *ChangeStoreStatusUseCase) fail(
	ctx context.Context,
	startedAt time.Time,
	failureCategory string,
	err error,
) (dto.StoreOutput, error) {
	uc.log(ctx, ports.LogEvent{
		Event:           "store.change_status.failed",
		Operation:       "change_store_status",
		FailureCategory: failureCategory,
	})

	uc.increment(ctx, ports.Metric{
		Name:  "store_change_status_failure_total",
		Value: 1,
		Labels: map[string]string{
			"failure_category": failureCategory,
		},
	})

	uc.observe(ctx, ports.Metric{
		Name:  "store_change_status_duration_seconds",
		Value: time.Since(startedAt).Seconds(),
	})

	return dto.StoreOutput{}, err
}

func (uc *ChangeStoreStatusUseCase) log(
	ctx context.Context,
	event ports.LogEvent,
) {
	if uc.logger == nil {
		return
	}

	_ = uc.logger.Log(ctx, event)
}

func (uc *ChangeStoreStatusUseCase) increment(
	ctx context.Context,
	metric ports.Metric,
) {
	if uc.metrics == nil {
		return
	}

	_ = uc.metrics.Increment(ctx, metric)
}

func (uc *ChangeStoreStatusUseCase) observe(
	ctx context.Context,
	metric ports.Metric,
) {
	if uc.metrics == nil {
		return
	}

	_ = uc.metrics.Observe(ctx, metric)
}
