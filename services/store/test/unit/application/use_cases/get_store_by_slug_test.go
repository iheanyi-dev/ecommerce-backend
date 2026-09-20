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

type getStoreBySlugRepositoryStub struct {
	findBySlugFn func(context.Context, string) (*entities.Store, error)
}

func (s *getStoreBySlugRepositoryStub) Create(
	context.Context,
	*entities.Store,
) error {
	return nil
}

func (s *getStoreBySlugRepositoryStub) Delete(
	context.Context,
	uuid.UUID,
) error {
	return nil
}

func (s *getStoreBySlugRepositoryStub) FindByID(
	context.Context,
	uuid.UUID,
) (*entities.Store, error) {
	return nil, nil
}

func (s *getStoreBySlugRepositoryStub) FindByOwnerID(
	context.Context,
	uuid.UUID,
) (*entities.Store, error) {
	return nil, nil
}

func (s *getStoreBySlugRepositoryStub) FindBySlug(
	ctx context.Context,
	slug string,
) (*entities.Store, error) {
	if s.findBySlugFn == nil {
		return nil, nil
	}

	return s.findBySlugFn(ctx, slug)
}

func (s *getStoreBySlugRepositoryStub) ListActive(
	ctx context.Context,
	query string,
	page int,
	pageSize int,
) ([]*entities.Store, int, error) {
	return nil, 0, nil
}

func (f *getStoreBySlugRepositoryStub) ExistsBySlug(_ context.Context, _ string) (bool, error) {
	return true, nil
}

func (s *getStoreBySlugRepositoryStub) Update(
	context.Context,
	*entities.Store,
) error {
	return nil
}

type getStoreBySlugLoggerStub struct {
	events []ports.LogEvent
}

func (s *getStoreBySlugLoggerStub) Log(
	_ context.Context,
	event ports.LogEvent,
) error {
	s.events = append(s.events, event)
	return nil
}

type getStoreBySlugMetricsStub struct {
	increments   []ports.Metric
	observations []ports.Metric
}

func (s *getStoreBySlugMetricsStub) Increment(
	_ context.Context,
	metric ports.Metric,
) error {
	s.increments = append(s.increments, metric)
	return nil
}

func (s *getStoreBySlugMetricsStub) Observe(
	_ context.Context,
	metric ports.Metric,
) error {
	s.observations = append(s.observations, metric)
	return nil
}

func newGetStoreBySlugApplication(
	repository ports.StoreRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *use_cases.GetStoreBySlugUseCase {
	return use_cases.NewGetStoreBySlugUseCase(
		repository,
		logger,
		metrics,
	)
}

func newStoreForSlugTest(t *testing.T) *entities.Store {
	t.Helper()

	store, err := entities.NewStore(
		uuid.New(),
		"My Store",
		"my-store",
		"My store description",
		nil,
		string(valueobjects.PlanTypeBasic),
	)
	require.NoError(t, err)

	return store
}

func TestGetStoreBySlugUseCase_Execute(t *testing.T) {
	t.Run("returns store", func(t *testing.T) {
		store := newStoreForSlugTest(t)

		repository := &getStoreBySlugRepositoryStub{
			findBySlugFn: func(
				_ context.Context,
				slug string,
			) (*entities.Store, error) {
				assert.Equal(t, store.Slug().Value(), slug)

				return store, nil
			},
		}

		logger := &getStoreBySlugLoggerStub{}
		metrics := &getStoreBySlugMetricsStub{}

		application := newGetStoreBySlugApplication(
			repository,
			logger,
			metrics,
		)

		output, err := application.Execute(
			context.Background(),
			store.Slug().Value(),
		)

		require.NoError(t, err)

		assert.Equal(t, store.ID(), output.ID)
		assert.Equal(t, store.OwnerID(), output.OwnerID)
		assert.Equal(t, store.Name(), output.Name)
		assert.Equal(t, store.Slug(), output.Slug)
		assert.Equal(t, store.Description(), output.Description)
		assert.Equal(t, store.ImageReference(), output.ImageReference)
		assert.Equal(t, store.Status(), output.Status)
		assert.Equal(t, store.Plan(), output.Plan)
		assert.Equal(t, store.CreatedAt(), output.CreatedAt)
		assert.Equal(t, store.UpdatedAt(), output.UpdatedAt)

		require.Len(t, logger.events, 1)
		assert.Equal(
			t,
			"store.get_by_slug.succeeded",
			logger.events[0].Event,
		)
		assert.Equal(
			t,
			"get_store_by_slug",
			logger.events[0].Operation,
		)

		require.GreaterOrEqual(t, len(metrics.increments), 2)
		assert.Equal(
			t,
			"store_get_by_slug_total",
			metrics.increments[0].Name,
		)
		assert.Equal(
			t,
			"store_get_by_slug_success_total",
			metrics.increments[1].Name,
		)

		require.Len(t, metrics.observations, 1)
		assert.Equal(
			t,
			"store_get_by_slug_duration_seconds",
			metrics.observations[0].Name,
		)
	})

	t.Run("returns not found for inactive store when unauthenticated", func(t *testing.T) {
		store := newStoreForSlugTest(t)

		require.NoError(t, store.Deactivate())

		repository := &getStoreBySlugRepositoryStub{
			findBySlugFn: func(
				context.Context,
				string,
			) (*entities.Store, error) {
				return store, nil
			},
		}

		application := newGetStoreBySlugApplication(
			repository,
			&getStoreBySlugLoggerStub{},
			&getStoreBySlugMetricsStub{},
		)

		output, err := application.Execute(
			context.Background(),
			store.Slug().Value(),
		)

		require.ErrorIs(t, err, applicationerrors.ErrStoreNotFound)
		assert.Nil(t, output)
	})

	t.Run("returns not found for inactive store when authenticated user is not owner", func(t *testing.T) {
		store := newStoreForSlugTest(t)

		require.NoError(t, store.Deactivate())

		repository := &getStoreBySlugRepositoryStub{
			findBySlugFn: func(
				context.Context,
				string,
			) (*entities.Store, error) {
				return store, nil
			},
		}

		ctx := ports.WithAuthenticatedIdentity(
			context.Background(),
			ports.AuthenticatedIdentity{
				UserID: uuid.New(),
				Role:   "user",
			},
		)

		application := newGetStoreBySlugApplication(
			repository,
			&getStoreBySlugLoggerStub{},
			&getStoreBySlugMetricsStub{},
		)

		output, err := application.Execute(
			ctx,
			store.Slug().Value(),
		)

		require.ErrorIs(t, err, applicationerrors.ErrStoreNotFound)
		assert.Nil(t, output)
	})

	t.Run("returns inactive store when authenticated user is owner", func(t *testing.T) {
		store := newStoreForSlugTest(t)

		require.NoError(t, store.Deactivate())

		repository := &getStoreBySlugRepositoryStub{
			findBySlugFn: func(
				context.Context,
				string,
			) (*entities.Store, error) {
				return store, nil
			},
		}

		ctx := ports.WithAuthenticatedIdentity(
			context.Background(),
			ports.AuthenticatedIdentity{
				UserID: store.OwnerID().Value(),
				Role:   "vendor",
			},
		)

		application := newGetStoreBySlugApplication(
			repository,
			&getStoreBySlugLoggerStub{},
			&getStoreBySlugMetricsStub{},
		)

		output, err := application.Execute(
			ctx,
			store.Slug().Value(),
		)

		require.NoError(t, err)
		require.NotNil(t, output)
		assert.Equal(t, store.ID(), output.ID)
		assert.Equal(t, valueobjects.StatusInactive, output.Status)
	})

	t.Run("returns not found when repository returns nil", func(t *testing.T) {
		repository := &getStoreBySlugRepositoryStub{
			findBySlugFn: func(
				context.Context,
				string,
			) (*entities.Store, error) {
				return nil, nil
			},
		}

		logger := &getStoreBySlugLoggerStub{}
		metrics := &getStoreBySlugMetricsStub{}

		application := newGetStoreBySlugApplication(
			repository,
			logger,
			metrics,
		)

		output, err := application.Execute(
			context.Background(),
			"missing-store",
		)

		require.ErrorIs(t, err, applicationerrors.ErrStoreNotFound)
		assert.Nil(t, output)
		require.Len(t, logger.events, 1)
		assert.Equal(
			t,
			"store.get_by_slug.failed",
			logger.events[0].Event,
		)
		assert.Equal(
			t,
			"get_store_by_slug",
			logger.events[0].Operation,
		)
		assert.Equal(
			t,
			"store_not_found",
			logger.events[0].FailureCategory,
		)

		require.Len(t, metrics.increments, 2)
		assert.Equal(
			t,
			"store_get_by_slug_total",
			metrics.increments[0].Name,
		)
		assert.Equal(
			t,
			"store_get_by_slug_failure_total",
			metrics.increments[1].Name,
		)
		assert.Equal(
			t,
			"store_not_found",
			metrics.increments[1].Labels["failure_category"],
		)

		require.Len(t, metrics.observations, 1)
		assert.Equal(
			t,
			"store_get_by_slug_duration_seconds",
			metrics.observations[0].Name,
		)
	})

	t.Run("returns repository error", func(t *testing.T) {
		expectedErr := errors.New("repository failure")

		repository := &getStoreBySlugRepositoryStub{
			findBySlugFn: func(
				context.Context,
				string,
			) (*entities.Store, error) {
				return nil, expectedErr
			},
		}

		logger := &getStoreBySlugLoggerStub{}
		metrics := &getStoreBySlugMetricsStub{}

		application := newGetStoreBySlugApplication(
			repository,
			logger,
			metrics,
		)

		output, err := application.Execute(
			context.Background(),
			"my-store",
		)

		require.ErrorIs(t, err, expectedErr)
		assert.Nil(t, output)
		require.Len(t, logger.events, 1)
		assert.Equal(
			t,
			"store.get_by_slug.failed",
			logger.events[0].Event,
		)
		assert.Equal(
			t,
			"store_lookup",
			logger.events[0].FailureCategory,
		)

		require.Len(t, metrics.increments, 2)
		assert.Equal(
			t,
			"store_get_by_slug_failure_total",
			metrics.increments[1].Name,
		)
		assert.Equal(
			t,
			"store_lookup",
			metrics.increments[1].Labels["failure_category"],
		)
	})

	t.Run("passes context to repository", func(t *testing.T) {
		store := newStoreForSlugTest(t)

		type contextKey struct{}

		ctx := context.WithValue(
			context.Background(),
			contextKey{},
			"test-value",
		)

		repository := &getStoreBySlugRepositoryStub{
			findBySlugFn: func(
				receivedCtx context.Context,
				_ string,
			) (*entities.Store, error) {
				assert.Equal(
					t,
					"test-value",
					receivedCtx.Value(contextKey{}),
				)

				return store, nil
			},
		}

		application := newGetStoreBySlugApplication(
			repository,
			&getStoreBySlugLoggerStub{},
			&getStoreBySlugMetricsStub{},
		)

		_, err := application.Execute(
			ctx,
			store.Slug().Value(),
		)

		require.NoError(t, err)
	})

	t.Run("does not mutate store", func(t *testing.T) {
		store := newStoreForSlugTest(t)
		beforeUpdatedAt := store.UpdatedAt()

		repository := &getStoreBySlugRepositoryStub{
			findBySlugFn: func(
				context.Context,
				string,
			) (*entities.Store, error) {
				return store, nil
			},
		}

		application := newGetStoreBySlugApplication(
			repository,
			&getStoreBySlugLoggerStub{},
			&getStoreBySlugMetricsStub{},
		)

		_, err := application.Execute(
			context.Background(),
			store.Slug().Value(),
		)

		require.NoError(t, err)
		assert.Equal(t, beforeUpdatedAt, store.UpdatedAt())
		assert.Equal(t, valueobjects.StatusActive, store.Status())
	})

	t.Run("does not depend on authentication", func(t *testing.T) {
		store := newStoreForSlugTest(t)

		repository := &getStoreBySlugRepositoryStub{
			findBySlugFn: func(
				context.Context,
				string,
			) (*entities.Store, error) {
				return store, nil
			},
		}

		application := newGetStoreBySlugApplication(
			repository,
			&getStoreBySlugLoggerStub{},
			&getStoreBySlugMetricsStub{},
		)

		output, err := application.Execute(
			context.Background(),
			store.Slug().Value(),
		)

		require.NoError(t, err)
		assert.Equal(t, store.ID(), output.ID)
	})
}
