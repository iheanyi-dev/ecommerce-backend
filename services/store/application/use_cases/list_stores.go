package use_cases

import (
	"context"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
)

// ListStoresUseCase retrieves a paginated list of active Stores.
//
// Store discovery is public, so this use case does not require an
// authenticated identity. Search and pagination are delegated to the
// repository so filtering and paging happen at the persistence layer rather
// than loading the complete Store collection into application memory.
type ListStoresUseCase struct {
	storeRepository ports.StoreRepository
	logger          ports.Logger
	metrics         ports.Metrics
}

func NewListStoresUseCase(
	storeRepository ports.StoreRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *ListStoresUseCase {
	return &ListStoresUseCase{
		storeRepository: storeRepository,
		logger:          logger,
		metrics:         metrics,
	}
}

// Execute returns one page of active Stores.
//
// Query is an optional partial name-search value. The repository is
// responsible for applying the actual database matching semantics.
func (uc *ListStoresUseCase) Execute(
	ctx context.Context,
	query string,
	page int,
	pageSize int,
) (*dto.ListStoresOutput, error) {
	startedAt := time.Now()

	uc.metrics.Increment(ctx, ports.Metric{
		Name:  "store_list_total",
		Value: 1,
	})

	if page < 1 || pageSize < 1 || pageSize > 100 {
		uc.logger.Log(ctx, ports.LogEvent{
			Event:           "store.list.failed",
			Operation:       "list_stores",
			FailureCategory: "invalid_pagination",
		})

		uc.metrics.Increment(ctx, ports.Metric{
			Name:  "store_list_failure_total",
			Value: 1,
			Labels: map[string]string{
				"failure_category": "invalid_pagination",
			},
		})

		uc.metrics.Observe(ctx, ports.Metric{
			Name:  "store_list_duration_seconds",
			Value: time.Since(startedAt).Seconds(),
		})

		return nil, applicationerrors.ErrInvalidPagination
	}

	stores, total, err := uc.storeRepository.ListActive(
		ctx,
		query,
		page,
		pageSize,
	)
	if err != nil {
		uc.logger.Log(ctx, ports.LogEvent{
			Event:           "store.list.failed",
			Operation:       "list_stores",
			FailureCategory: "store_lookup",
		})

		uc.metrics.Increment(ctx, ports.Metric{
			Name:  "store_list_failure_total",
			Value: 1,
			Labels: map[string]string{
				"failure_category": "store_lookup",
			},
		})

		uc.metrics.Observe(ctx, ports.Metric{
			Name:  "store_list_duration_seconds",
			Value: time.Since(startedAt).Seconds(),
		})

		return nil, err
	}

	output := &dto.ListStoresOutput{
		Stores:     make([]dto.StoreOutput, 0, len(stores)),
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: calculateTotalPages(total, pageSize),
	}

	for _, store := range stores {
		if store == nil {
			continue
		}

		output.Stores = append(
			output.Stores,
			toStoreOutput(store),
		)
	}

	uc.logger.Log(ctx, ports.LogEvent{
		Event:     "store.list.succeeded",
		Operation: "list_stores",
	})

	uc.metrics.Increment(ctx, ports.Metric{
		Name:  "store_list_success_total",
		Value: 1,
	})

	uc.metrics.Observe(ctx, ports.Metric{
		Name:  "store_list_duration_seconds",
		Value: time.Since(startedAt).Seconds(),
	})

	return output, nil
}

func calculateTotalPages(total, pageSize int) int {
	if total == 0 {
		return 0
	}

	return (total + pageSize - 1) / pageSize
}

// boundary remains obvious when reviewing the application layer.
