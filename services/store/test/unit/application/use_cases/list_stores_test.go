package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
)

type listStoresRepositoryStub struct {
	listActiveFn func(
		context.Context,
		string,
		int,
		int,
	) ([]*entities.Store, int, error)
}

func (s *listStoresRepositoryStub) Create(
	context.Context,
	*entities.Store,
) error {
	return nil
}

func (s *listStoresRepositoryStub) Delete(
	context.Context,
	uuid.UUID,
) error {
	return nil
}

func (s *listStoresRepositoryStub) FindByID(
	context.Context,
	uuid.UUID,
) (*entities.Store, error) {
	return nil, nil
}

func (s *listStoresRepositoryStub) FindByOwnerID(
	context.Context,
	uuid.UUID,
) (*entities.Store, error) {
	return nil, nil
}

func (s *listStoresRepositoryStub) FindBySlug(
	context.Context,
	string,
) (*entities.Store, error) {
	return nil, nil
}
func (f *listStoresRepositoryStub) ExistsBySlug(_ context.Context, _ string) (bool, error) {
	return true, nil
}
func (s *listStoresRepositoryStub) ListActive(
	ctx context.Context,
	query string,
	page int,
	pageSize int,
) ([]*entities.Store, int, error) {
	if s.listActiveFn == nil {
		return nil, 0, nil
	}

	return s.listActiveFn(ctx, query, page, pageSize)
}

func (s *listStoresRepositoryStub) Update(
	context.Context,
	*entities.Store,
) error {
	return nil
}

type listStoresLoggerStub struct {
	events []ports.LogEvent
}

func (s *listStoresLoggerStub) Log(
	_ context.Context,
	event ports.LogEvent,
) error {
	s.events = append(s.events, event)
	return nil
}

type listStoresMetricsStub struct {
	increments   []ports.Metric
	observations []ports.Metric
}

func (s *listStoresMetricsStub) Increment(
	_ context.Context,
	metric ports.Metric,
) error {
	s.increments = append(s.increments, metric)
	return nil
}

func (s *listStoresMetricsStub) Observe(
	_ context.Context,
	metric ports.Metric,
) error {
	s.observations = append(s.observations, metric)
	return nil
}

func newListStoresApplication(
	repository ports.StoreRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *use_cases.ListStoresUseCase {
	return use_cases.NewListStoresUseCase(
		repository,
		logger,
		metrics,
	)
}

func newStoreForListTest(t *testing.T, name string) *entities.Store {
	t.Helper()

	store, err := entities.NewStore(
		uuid.New(),
		name,
		"store-"+uuid.NewString(),
		"Store description",
		nil,
		string(valueobjects.PlanTypeBasic),
	)
	require.NoError(t, err)

	return store
}

func TestListStoresUseCase_Execute(t *testing.T) {
	t.Run("returns paginated stores and metadata", func(t *testing.T) {
		storeOne := newStoreForListTest(t, "Phone World")
		storeTwo := newStoreForListTest(t, "Smart Phones NG")

		repository := &listStoresRepositoryStub{
			listActiveFn: func(
				_ context.Context,
				query string,
				page int,
				pageSize int,
			) ([]*entities.Store, int, error) {
				assert.Equal(t, "phone", query)
				assert.Equal(t, 2, page)
				assert.Equal(t, 10, pageSize)

				return []*entities.Store{
					storeOne,
					storeTwo,
				}, 22, nil
			},
		}

		logger := &listStoresLoggerStub{}
		metrics := &listStoresMetricsStub{}

		application := newListStoresApplication(
			repository,
			logger,
			metrics,
		)

		output, err := application.Execute(
			context.Background(),
			"phone",
			2,
			10,
		)

		require.NoError(t, err)
		require.NotNil(t, output)

		assert.Equal(t, 2, len(output.Stores))
		assert.Equal(t, storeOne.ID(), output.Stores[0].ID)
		assert.Equal(t, storeTwo.ID(), output.Stores[1].ID)
		assert.Equal(t, 2, output.Page)
		assert.Equal(t, 10, output.PageSize)
		assert.Equal(t, 22, output.Total)
		assert.Equal(t, 3, output.TotalPages)

		require.Len(t, logger.events, 1)
		assert.Equal(
			t,
			"store.list.succeeded",
			logger.events[0].Event,
		)
		assert.Equal(
			t,
			"list_stores",
			logger.events[0].Operation,
		)

		require.GreaterOrEqual(t, len(metrics.increments), 2)
		assert.Equal(
			t,
			"store_list_total",
			metrics.increments[0].Name,
		)
		assert.Equal(
			t,
			"store_list_success_total",
			metrics.increments[1].Name,
		)

		require.Len(t, metrics.observations, 1)
		assert.Equal(
			t,
			"store_list_duration_seconds",
			metrics.observations[0].Name,
		)
	})

	t.Run("returns empty page successfully", func(t *testing.T) {
		repository := &listStoresRepositoryStub{
			listActiveFn: func(
				_ context.Context,
				_ string,
				_ int,
				_ int,
			) ([]*entities.Store, int, error) {
				return []*entities.Store{}, 0, nil
			},
		}

		application := newListStoresApplication(
			repository,
			&listStoresLoggerStub{},
			&listStoresMetricsStub{},
		)

		output, err := application.Execute(
			context.Background(),
			"missing",
			1,
			20,
		)

		require.NoError(t, err)
		require.NotNil(t, output)
		assert.Empty(t, output.Stores)
		assert.Equal(t, 1, output.Page)
		assert.Equal(t, 20, output.PageSize)
		assert.Equal(t, 0, output.Total)
		assert.Equal(t, 0, output.TotalPages)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		expectedErr := errors.New("repository failure")

		repository := &listStoresRepositoryStub{
			listActiveFn: func(
				context.Context,
				string,
				int,
				int,
			) ([]*entities.Store, int, error) {
				return nil, 0, expectedErr
			},
		}

		logger := &listStoresLoggerStub{}
		metrics := &listStoresMetricsStub{}

		application := newListStoresApplication(
			repository,
			logger,
			metrics,
		)

		output, err := application.Execute(
			context.Background(),
			"phone",
			1,
			20,
		)

		require.ErrorIs(t, err, expectedErr)
		assert.Nil(t, output)

		require.Len(t, logger.events, 1)
		assert.Equal(
			t,
			"store.list.failed",
			logger.events[0].Event,
		)
		assert.Equal(
			t,
			"list_stores",
			logger.events[0].Operation,
		)
		assert.Equal(
			t,
			"store_lookup",
			logger.events[0].FailureCategory,
		)
	})

	t.Run("passes context to repository", func(t *testing.T) {
		type contextKey struct{}

		ctx := context.WithValue(
			context.Background(),
			contextKey{},
			"test-value",
		)

		repository := &listStoresRepositoryStub{
			listActiveFn: func(
				receivedCtx context.Context,
				_ string,
				_ int,
				_ int,
			) ([]*entities.Store, int, error) {
				assert.Equal(
					t,
					"test-value",
					receivedCtx.Value(contextKey{}),
				)

				return []*entities.Store{}, 0, nil
			},
		}

		application := newListStoresApplication(
			repository,
			&listStoresLoggerStub{},
			&listStoresMetricsStub{},
		)

		_, err := application.Execute(
			ctx,
			"",
			1,
			20,
		)

		require.NoError(t, err)
	})

	t.Run("rejects page less than one", func(t *testing.T) {
		called := false

		repository := &listStoresRepositoryStub{
			listActiveFn: func(
				context.Context,
				string,
				int,
				int,
			) ([]*entities.Store, int, error) {
				called = true
				return nil, 0, nil
			},
		}

		application := newListStoresApplication(
			repository,
			&listStoresLoggerStub{},
			&listStoresMetricsStub{},
		)

		output, err := application.Execute(
			context.Background(),
			"",
			0,
			20,
		)

		require.ErrorIs(t, err, applicationerrors.ErrInvalidPagination)
		assert.Nil(t, output)
		assert.False(t, called)
	})

	t.Run("rejects page size less than one", func(t *testing.T) {
		called := false

		repository := &listStoresRepositoryStub{
			listActiveFn: func(
				context.Context,
				string,
				int,
				int,
			) ([]*entities.Store, int, error) {
				called = true
				return nil, 0, nil
			},
		}

		application := newListStoresApplication(
			repository,
			&listStoresLoggerStub{},
			&listStoresMetricsStub{},
		)

		output, err := application.Execute(
			context.Background(),
			"",
			1,
			0,
		)

		require.ErrorIs(t, err, applicationerrors.ErrInvalidPagination)
		assert.Nil(t, output)
		assert.False(t, called)
	})

	t.Run("rejects page size greater than one hundred", func(t *testing.T) {
		called := false

		repository := &listStoresRepositoryStub{
			listActiveFn: func(
				context.Context,
				string,
				int,
				int,
			) ([]*entities.Store, int, error) {
				called = true
				return nil, 0, nil
			},
		}

		application := newListStoresApplication(
			repository,
			&listStoresLoggerStub{},
			&listStoresMetricsStub{},
		)

		output, err := application.Execute(
			context.Background(),
			"",
			1,
			101,
		)

		require.ErrorIs(t, err, applicationerrors.ErrInvalidPagination)
		assert.Nil(t, output)
		assert.False(t, called)
	})

	t.Run("does not mutate stores", func(t *testing.T) {
		store := newStoreForListTest(t, "Phone World")
		beforeUpdatedAt := store.UpdatedAt()
		beforeStatus := store.Status()

		repository := &listStoresRepositoryStub{
			listActiveFn: func(
				context.Context,
				string,
				int,
				int,
			) ([]*entities.Store, int, error) {
				return []*entities.Store{store}, 1, nil
			},
		}

		application := newListStoresApplication(
			repository,
			&listStoresLoggerStub{},
			&listStoresMetricsStub{},
		)

		_, err := application.Execute(
			context.Background(),
			"",
			1,
			20,
		)

		require.NoError(t, err)
		assert.Equal(t, beforeUpdatedAt, store.UpdatedAt())
		assert.Equal(t, beforeStatus, store.Status())
	})
}
