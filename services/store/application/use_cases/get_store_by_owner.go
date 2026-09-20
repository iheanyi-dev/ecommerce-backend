package use_cases

import (
	"context"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
)

type GetStoreByOwnerUseCase struct {
	storeRepository ports.StoreRepository
	logger          ports.Logger
	metrics         ports.Metrics
}

func NewGetStoreByOwnerUseCase(
	storeRepository ports.StoreRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *GetStoreByOwnerUseCase {
	return &GetStoreByOwnerUseCase{
		storeRepository: storeRepository,
		logger:          logger,
		metrics:         metrics,
	}
}

// Execute retrieves the store owned by the authenticated user.
//
// The owner ID is always taken from the authenticated identity in the
// request context. The caller cannot supply another owner ID.
func (uc *GetStoreByOwnerUseCase) Execute(
	ctx context.Context,
) (*dto.StoreOutput, error) {
	startedAt := time.Now()

	identity, ok := ports.AuthenticatedIdentityFromContext(ctx)
	if !ok {
		uc.logger.Log(ctx, ports.LogEvent{
			Event:           "store.get_by_owner.failed",
			Operation:       "get_store_by_owner",
			FailureCategory: "unauthenticated",
		})
		uc.metrics.Increment(ctx, ports.Metric{
			Name:  "store_get_by_owner_failure_total",
			Value: 1,
			Labels: map[string]string{
				"failure_category": "unauthenticated",
			},
		})
		uc.metrics.Observe(ctx, ports.Metric{
			Name:  "store_get_by_owner_duration_seconds",
			Value: time.Since(startedAt).Seconds(),
		})

		return nil, applicationerrors.ErrUnauthenticated
	}

	store, err := uc.storeRepository.FindByOwnerID(ctx, identity.UserID)
	if err != nil {
		uc.logger.Log(ctx, ports.LogEvent{
			Event:           "store.get_by_owner.failed",
			Operation:       "get_store_by_owner",
			UserID:          identity.UserID.String(),
			Role:            identity.Role,
			FailureCategory: "store_lookup",
		})
		uc.metrics.Increment(ctx, ports.Metric{
			Name:  "store_get_by_owner_failure_total",
			Value: 1,
			Labels: map[string]string{
				"failure_category": "store_lookup",
			},
		})
		uc.metrics.Observe(ctx, ports.Metric{
			Name:  "store_get_by_owner_duration_seconds",
			Value: time.Since(startedAt).Seconds(),
		})

		return nil, err
	}

	if store == nil {
		uc.logger.Log(ctx, ports.LogEvent{
			Event:           "store.get_by_owner.failed",
			Operation:       "get_store_by_owner",
			UserID:          identity.UserID.String(),
			Role:            identity.Role,
			FailureCategory: "store_not_found",
		})
		uc.metrics.Increment(ctx, ports.Metric{
			Name:  "store_get_by_owner_failure_total",
			Value: 1,
			Labels: map[string]string{
				"failure_category": "store_not_found",
			},
		})
		uc.metrics.Observe(ctx, ports.Metric{
			Name:  "store_get_by_owner_duration_seconds",
			Value: time.Since(startedAt).Seconds(),
		})

		return nil, applicationerrors.ErrStoreNotFound
	}
	uc.logger.Log(ctx, ports.LogEvent{
		Event:     "store.get_by_owner.succeeded",
		Operation: "get_store_by_owner",
		UserID:    identity.UserID.String(),
		Role:      identity.Role,
	})

	uc.metrics.Increment(ctx, ports.Metric{
		Name:  "store_get_by_owner_total",
		Value: 1,
	})
	uc.metrics.Increment(ctx, ports.Metric{
		Name:  "store_get_by_owner_success_total",
		Value: 1,
	})
	uc.metrics.Observe(ctx, ports.Metric{
		Name:  "store_get_by_owner_duration_seconds",
		Value: time.Since(startedAt).Seconds(),
	})

	output := toStoreOutput(store)

	return &output, nil
}
