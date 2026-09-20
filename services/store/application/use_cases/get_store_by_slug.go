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

type GetStoreBySlugUseCase struct {
	storeRepository ports.StoreRepository
	logger          ports.Logger
	metrics         ports.Metrics
}

func NewGetStoreBySlugUseCase(
	storeRepository ports.StoreRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *GetStoreBySlugUseCase {
	return &GetStoreBySlugUseCase{
		storeRepository: storeRepository,
		logger:          logger,
		metrics:         metrics,
	}
}

func (uc *GetStoreBySlugUseCase) Execute(
	ctx context.Context,
	slug string,
) (*dto.StoreOutput, error) {
	startedAt := time.Now()

	uc.metrics.Increment(ctx, ports.Metric{
		Name:  "store_get_by_slug_total",
		Value: 1,
	})

	store, err := uc.storeRepository.FindBySlug(ctx, slug)
	if err != nil {
		uc.logger.Log(ctx, ports.LogEvent{
			Event:           "store.get_by_slug.failed",
			Operation:       "get_store_by_slug",
			FailureCategory: "store_lookup",
		})

		uc.metrics.Increment(ctx, ports.Metric{
			Name:  "store_get_by_slug_failure_total",
			Value: 1,
			Labels: map[string]string{
				"failure_category": "store_lookup",
			},
		})

		uc.metrics.Observe(ctx, ports.Metric{
			Name:  "store_get_by_slug_duration_seconds",
			Value: time.Since(startedAt).Seconds(),
		})

		return nil, err
	}

	if store == nil {
		uc.logger.Log(ctx, ports.LogEvent{
			Event:           "store.get_by_slug.failed",
			Operation:       "get_store_by_slug",
			FailureCategory: "store_not_found",
		})

		uc.metrics.Increment(ctx, ports.Metric{
			Name:  "store_get_by_slug_failure_total",
			Value: 1,
			Labels: map[string]string{
				"failure_category": "store_not_found",
			},
		})

		uc.metrics.Observe(ctx, ports.Metric{
			Name:  "store_get_by_slug_duration_seconds",
			Value: time.Since(startedAt).Seconds(),
		})

		return nil, applicationerrors.ErrStoreNotFound
	}

	if !canViewStore(ctx, store.Status(), store.IsOwnedBy) {
		uc.logger.Log(ctx, ports.LogEvent{
			Event:           "store.get_by_slug.failed",
			Operation:       "get_store_by_slug",
			FailureCategory: "store_not_visible",
		})

		uc.metrics.Increment(ctx, ports.Metric{
			Name:  "store_get_by_slug_failure_total",
			Value: 1,
			Labels: map[string]string{
				"failure_category": "store_not_visible",
			},
		})

		uc.metrics.Observe(ctx, ports.Metric{
			Name:  "store_get_by_slug_duration_seconds",
			Value: time.Since(startedAt).Seconds(),
		})

		return nil, applicationerrors.ErrStoreNotFound
	}

	output := toStoreOutput(store)

	uc.logger.Log(ctx, ports.LogEvent{
		Event:     "store.get_by_slug.succeeded",
		Operation: "get_store_by_slug",
	})

	uc.metrics.Increment(ctx, ports.Metric{
		Name:  "store_get_by_slug_success_total",
		Value: 1,
	})

	uc.metrics.Observe(ctx, ports.Metric{
		Name:  "store_get_by_slug_duration_seconds",
		Value: time.Since(startedAt).Seconds(),
	})

	return &output, nil
}

func canViewStore(
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
