package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	usecases "github.com/iheanyi-dev/ecommerce-backend/services/store/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/test/unit/fixtures"
)

func TestGetStoreByOwnerUseCase_ReturnsStore(t *testing.T) {
	ownerID := uuid.New()
	storeID := uuid.New()

	store, err := entities.ReconstituteStore(
		storeID,
		ownerID,
		"Test Store",
		"test-store",
		"Test store description",
		nil,
		string(valueobjects.PlanTypeBasic),
		string(valueobjects.StatusActive),
		testTimestamp(),
		testTimestamp(),
	)
	require.NoError(t, err)

	repo := &fixtures.MockStoreRepository{
		Store: store,
	}
	logger := &fakeGetStoreByOwnerLogger{}
	metrics := &fakeGetStoreByOwnerMetrics{}

	ctx := ports.WithAuthenticatedIdentity(
		context.Background(),
		ports.AuthenticatedIdentity{
			UserID: ownerID,
			Role:   "vendor",
		},
	)

	uc := usecases.NewGetStoreByOwnerUseCase(
		repo,
		logger,
		metrics,
	)

	output, err := uc.Execute(ctx)

	require.NoError(t, err)
	require.NotNil(t, output)

	require.Equal(t, store.ID(), output.ID)
	require.Equal(t, store.OwnerID(), output.OwnerID)
	require.Equal(t, store.Name(), output.Name)
	require.Equal(t, store.Slug(), output.Slug)
	require.Equal(t, store.Description(), output.Description)
	require.Equal(t, store.ImageReference(), output.ImageReference)
	require.Equal(t, store.Status(), output.Status)
	require.Equal(t, store.Plan(), output.Plan)
	require.Equal(t, store.CreatedAt(), output.CreatedAt)
	require.Equal(t, store.UpdatedAt(), output.UpdatedAt)

	require.Equal(t, ownerID, repo.OwnerID)
	require.True(t, repo.FindByOwnerIDCalled)

	require.Len(t, logger.events, 1)
	require.Equal(
		t,
		"store.get_by_owner.succeeded",
		logger.events[0].Event,
	)
	require.Equal(
		t,
		"get_store_by_owner",
		logger.events[0].Operation,
	)
	require.Equal(t, ownerID.String(), logger.events[0].UserID)
	require.Equal(t, "vendor", logger.events[0].Role)

	require.Contains(t, metrics.incremented, ports.Metric{
		Name:  "store_get_by_owner_total",
		Value: 1,
	})
	require.Contains(t, metrics.incremented, ports.Metric{
		Name:  "store_get_by_owner_success_total",
		Value: 1,
	})
	require.Len(t, metrics.observed, 1)
	require.Equal(
		t,
		"store_get_by_owner_duration_seconds",
		metrics.observed[0].Name,
	)
}

func TestGetStoreByOwnerUseCase_RejectsUnauthenticatedRequest(t *testing.T) {
	repo := &fixtures.MockStoreRepository{}
	logger := &fakeGetStoreByOwnerLogger{}
	metrics := &fakeGetStoreByOwnerMetrics{}

	uc := usecases.NewGetStoreByOwnerUseCase(
		repo,
		logger,
		metrics,
	)

	output, err := uc.Execute(context.Background())

	require.ErrorIs(t, err, applicationerrors.ErrUnauthenticated)
	require.Nil(t, output)

	require.False(t, repo.FindByOwnerIDCalled)

	require.Len(t, logger.events, 1)
	require.Equal(
		t,
		"store.get_by_owner.failed",
		logger.events[0].Event,
	)
	require.Equal(
		t,
		"unauthenticated",
		logger.events[0].FailureCategory,
	)
}

func TestGetStoreByOwnerUseCase_ReturnsNotFoundWhenOwnerHasNoStore(
	t *testing.T,
) {
	ownerID := uuid.New()

	repo := &fixtures.MockStoreRepository{}
	logger := &fakeGetStoreByOwnerLogger{}
	metrics := &fakeGetStoreByOwnerMetrics{}

	ctx := ports.WithAuthenticatedIdentity(
		context.Background(),
		ports.AuthenticatedIdentity{
			UserID: ownerID,
			Role:   "vendor",
		},
	)

	uc := usecases.NewGetStoreByOwnerUseCase(
		repo,
		logger,
		metrics,
	)

	output, err := uc.Execute(ctx)

	require.ErrorIs(t, err, applicationerrors.ErrStoreNotFound)
	require.Nil(t, output)

	require.True(t, repo.FindByOwnerIDCalled)

	require.Len(t, logger.events, 1)
	require.Equal(
		t,
		"store.get_by_owner.failed",
		logger.events[0].Event,
	)
	require.Equal(
		t,
		"store_not_found",
		logger.events[0].FailureCategory,
	)
}

func TestGetStoreByOwnerUseCase_PropagatesRepositoryError(t *testing.T) {
	ownerID := uuid.New()
	repositoryError := errors.New("repository failure")

	repo := &fixtures.MockStoreRepository{
		FindByOwnerIDErr: repositoryError,
	}
	logger := &fakeGetStoreByOwnerLogger{}
	metrics := &fakeGetStoreByOwnerMetrics{}

	ctx := ports.WithAuthenticatedIdentity(
		context.Background(),
		ports.AuthenticatedIdentity{
			UserID: ownerID,
			Role:   "vendor",
		},
	)

	uc := usecases.NewGetStoreByOwnerUseCase(
		repo,
		logger,
		metrics,
	)

	output, err := uc.Execute(ctx)

	require.ErrorIs(t, err, repositoryError)
	require.Nil(t, output)

	require.True(t, repo.FindByOwnerIDCalled)

	require.Len(t, logger.events, 1)
	require.Equal(
		t,
		"store.get_by_owner.failed",
		logger.events[0].Event,
	)
	require.Equal(
		t,
		"store_lookup",
		logger.events[0].FailureCategory,
	)
}

type fakeGetStoreByOwnerLogger struct {
	events []ports.LogEvent
}

func (f *fakeGetStoreByOwnerLogger) Log(
	_ context.Context,
	event ports.LogEvent,
) error {
	f.events = append(f.events, event)

	return nil
}

type fakeGetStoreByOwnerMetrics struct {
	incremented []ports.Metric
	observed    []ports.Metric
}

func (f *fakeGetStoreByOwnerMetrics) Increment(
	_ context.Context,
	metric ports.Metric,
) error {
	f.incremented = append(f.incremented, metric)

	return nil
}

func (f *fakeGetStoreByOwnerMetrics) Observe(
	_ context.Context,
	metric ports.Metric,
) error {
	f.observed = append(f.observed, metric)

	return nil
}
