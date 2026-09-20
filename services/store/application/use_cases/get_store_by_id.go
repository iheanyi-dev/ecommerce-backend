package use_cases

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
)

// GetStoreByIDUseCase retrieves a Store using its unique identifier.
//
// Store lookup by ID is public for active Stores. An inactive Store may only
// be viewed by its owner.
type GetStoreByIDUseCase struct {
	storeRepository ports.StoreRepository
	logger          ports.Logger
	metrics         ports.Metrics
}

// NewGetStoreByIDUseCase constructs the Get Store by ID use case with its
// application-layer capabilities.
func NewGetStoreByIDUseCase(
	storeRepository ports.StoreRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *GetStoreByIDUseCase {
	return &GetStoreByIDUseCase{
		storeRepository: storeRepository,
		logger:          logger,
		metrics:         metrics,
	}
}

// Execute retrieves a Store by its unique identifier.
//
// Visibility rules:
//
//   - Active Stores are publicly visible.
//   - Inactive Stores are visible only to their owner.
//   - An unauthenticated caller cannot view an inactive Store.
//   - An authenticated non-owner cannot view an inactive Store.
//
// An inaccessible inactive Store is reported as not found so that the
// application does not disclose the existence of a private/inactive Store.
func (uc *GetStoreByIDUseCase) Execute(
	ctx context.Context,
	storeID uuid.UUID,
) (dto.StoreOutput, error) {
	startedAt := time.Now()

	uc.increment(ctx, ports.Metric{
		Name:  "store_get_by_id_total",
		Value: 1,
	})

	store, err := uc.storeRepository.FindByID(ctx, storeID)
	if err != nil {
		return uc.fail(
			ctx,
			startedAt,
			"store_lookup",
			err,
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

	if !uc.canViewStore(ctx, store.Status(), store.IsOwnedBy) {
		return uc.fail(
			ctx,
			startedAt,
			"store_not_visible",
			applicationerrors.ErrStoreNotFound,
		)
	}

	output := toStoreOutput(store)

	uc.log(ctx, ports.LogEvent{
		Event:     "store.get_by_id.succeeded",
		Operation: "get_store_by_id",
	})

	uc.increment(ctx, ports.Metric{
		Name:  "store_get_by_id_success_total",
		Value: 1,
	})

	uc.observe(ctx, ports.Metric{
		Name:  "store_get_by_id_duration_seconds",
		Value: time.Since(startedAt).Seconds(),
	})

	return output, nil
}

// canViewStore applies the Store visibility policy.
//
// Active Stores are public. Inactive Stores require an authenticated identity
// belonging to the Store owner.
func (uc *GetStoreByIDUseCase) canViewStore(
	ctx context.Context,
	status valueobjects.Status,
	isOwnedBy func(uuid.UUID) bool,
) bool {
	if status == valueobjects.StatusActive {
		return true
	}

	identity, ok := ports.AuthenticatedIdentityFromContext(ctx)
	if !ok || identity.UserID == uuid.Nil {
		return false
	}

	return isOwnedBy(identity.UserID)
}

// fail centralizes failure observability so every failure path emits the same
// application-level logging, metric, and duration information.
func (uc *GetStoreByIDUseCase) fail(
	ctx context.Context,
	startedAt time.Time,
	failureCategory string,
	err error,
) (dto.StoreOutput, error) {
	uc.log(ctx, ports.LogEvent{
		Event:           "store.get_by_id.failed",
		Operation:       "get_store_by_id",
		FailureCategory: failureCategory,
	})

	uc.increment(ctx, ports.Metric{
		Name:  "store_get_by_id_failure_total",
		Value: 1,
		Labels: map[string]string{
			"failure_category": failureCategory,
		},
	})

	uc.observe(ctx, ports.Metric{
		Name:  "store_get_by_id_duration_seconds",
		Value: time.Since(startedAt).Seconds(),
	})

	return dto.StoreOutput{}, err
}

// log emits an observability event without allowing an observability failure
// to change the business result of the use case.
func (uc *GetStoreByIDUseCase) log(
	ctx context.Context,
	event ports.LogEvent,
) {
	_ = uc.logger.Log(ctx, event)
}

// increment emits a metric without allowing a metrics backend failure to
// change the business result of the use case.
func (uc *GetStoreByIDUseCase) increment(
	ctx context.Context,
	metric ports.Metric,
) {
	_ = uc.metrics.Increment(ctx, metric)
}

// observe records a duration metric without allowing an observability backend
// failure to change the business result of the use case.
func (uc *GetStoreByIDUseCase) observe(
	ctx context.Context,
	metric ports.Metric,
) {
	_ = uc.metrics.Observe(ctx, metric)
}
