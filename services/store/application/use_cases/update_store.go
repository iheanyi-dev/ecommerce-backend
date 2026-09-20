package use_cases

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
)

// UpdateStoreUseCase updates the mutable Store profile fields owned by the
// authenticated Store owner.
//
// Owners may change:
//   - name
//   - slug
//   - description
//
// Ownership, status, plan, image reference, and owner identity are not
// directly changed by this workflow.
type UpdateStoreUseCase struct {
	storeRepository ports.StoreRepository
	logger          ports.Logger
	metrics         ports.Metrics
}

// NewUpdateStoreUseCase constructs the Update Store use case.
func NewUpdateStoreUseCase(
	storeRepository ports.StoreRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *UpdateStoreUseCase {
	return &UpdateStoreUseCase{
		storeRepository: storeRepository,
		logger:          logger,
		metrics:         metrics,
	}
}

// Execute updates the Store belonging to the authenticated caller.
func (uc *UpdateStoreUseCase) Execute(
	ctx context.Context,
	input dto.UpdateStoreInput,
) (dto.UpdateStoreOutput, error) {
	startedAt := time.Now()

	uc.increment(ctx, ports.Metric{
		Name:  "store_update_total",
		Value: 1,
	})

	identity, ok := ports.AuthenticatedIdentityFromContext(ctx)
	if !ok || identity.UserID == uuid.Nil {
		return uc.fail(
			ctx,
			startedAt,
			"unauthenticated",
			applicationerrors.ErrUnauthenticated,
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

	if !store.IsOwnedBy(identity.UserID) {
		// Do not reveal whether a Store exists to a different authenticated
		// user. Ownership failure is therefore represented as not found.
		return uc.fail(
			ctx,
			startedAt,
			"store_not_found",
			applicationerrors.ErrStoreNotFound,
		)
	}

	// Check slug uniqueness only when the owner is actually changing it.
	currentSlug := store.Slug().Value()

	if input.Slug != currentSlug {
		exists, err := uc.storeRepository.ExistsBySlug(ctx, input.Slug)
		if err != nil {
			return uc.fail(
				ctx,
				startedAt,
				"slug_lookup",
				fmt.Errorf("check store slug: %w", err),
			)
		}

		if exists {
			return uc.fail(
				ctx,
				startedAt,
				"store_slug_already_exists",
				applicationerrors.ErrStoreSlugAlreadyExists,
			)
		}
	}

	// Mutate the aggregate through its domain behavior rather than changing
	// its internal fields from the application layer.
	if err := store.ChangeName(input.Name); err != nil {
		return uc.fail(
			ctx,
			startedAt,
			"store_validation",
			err,
		)
	}

	if input.Slug != currentSlug {
		if err := store.ChangeSlug(input.Slug); err != nil {
			return uc.fail(
				ctx,
				startedAt,
				"store_validation",
				err,
			)
		}
	}

	if err := store.ChangeDescription(input.Description); err != nil {
		return uc.fail(
			ctx,
			startedAt,
			"store_validation",
			err,
		)
	}

	if err := uc.storeRepository.Update(ctx, store); err != nil {
		return uc.fail(
			ctx,
			startedAt,
			"store_persistence",
			fmt.Errorf("update store: %w", err),
		)
	}

	output := toStoreOutput(store)

	uc.log(ctx, ports.LogEvent{
		Event:     "store.update.succeeded",
		Operation: "update_store",
		UserID:    identity.UserID.String(),
		Role:      identity.Role,
	})

	uc.increment(ctx, ports.Metric{
		Name:  "store_update_success_total",
		Value: 1,
	})

	uc.observe(ctx, ports.Metric{
		Name:  "store_update_duration_seconds",
		Value: time.Since(startedAt).Seconds(),
	})

	return output, nil
}

func (uc *UpdateStoreUseCase) fail(
	ctx context.Context,
	startedAt time.Time,
	failureCategory string,
	err error,
) (dto.UpdateStoreOutput, error) {
	uc.log(ctx, ports.LogEvent{
		Event:           "store.update.failed",
		Operation:       "update_store",
		FailureCategory: failureCategory,
	})

	uc.increment(ctx, ports.Metric{
		Name:  "store_update_failure_total",
		Value: 1,
		Labels: map[string]string{
			"failure_category": failureCategory,
		},
	})

	uc.observe(ctx, ports.Metric{
		Name:  "store_update_duration_seconds",
		Value: time.Since(startedAt).Seconds(),
	})

	return dto.UpdateStoreOutput{}, err
}

func (uc *UpdateStoreUseCase) log(
	ctx context.Context,
	event ports.LogEvent,
) {
	_ = uc.logger.Log(ctx, event)
}

func (uc *UpdateStoreUseCase) increment(
	ctx context.Context,
	metric ports.Metric,
) {
	_ = uc.metrics.Increment(ctx, metric)
}

func (uc *UpdateStoreUseCase) observe(
	ctx context.Context,
	metric ports.Metric,
) {
	_ = uc.metrics.Observe(ctx, metric)
}
