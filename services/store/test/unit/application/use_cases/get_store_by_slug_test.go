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
	"github.com/iheanyi-dev/ecommerce-backend/services/store/test/unit/fixtures"
)

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

		repository := &fixtures.MockStoreRepository{
			Store: store,
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

		repository := &fixtures.MockStoreRepository{
			Store: store,
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

		repository := &fixtures.MockStoreRepository{
			Store: store,
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

		repository := &fixtures.MockStoreRepository{
			Store: store,
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
		repository := &fixtures.MockStoreRepository{
			FindBySlugErr: applicationerrors.ErrStoreNotFound,
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
			"store_lookup",
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
			"store_lookup",
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

		repository := &fixtures.MockStoreRepository{
			FindBySlugErr: expectedErr,
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

	t.Run("does not mutate store", func(t *testing.T) {
		store := newStoreForSlugTest(t)
		beforeUpdatedAt := store.UpdatedAt()

		repository := &fixtures.MockStoreRepository{
			Store: store,
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

		repository := &fixtures.MockStoreRepository{
			Store: store,
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
