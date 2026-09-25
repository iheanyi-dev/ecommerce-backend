package use_cases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	usecases "github.com/iheanyi-dev/ecommerce-backend/services/store/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/test/unit/fixtures"
)

func TestGetStoreByIDUseCase_ReturnsStore(t *testing.T) {
	storeID := uuid.New()
	ownerID := uuid.New()

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
	logger := &fakeGetStoreByIDLogger{}
	metrics := &fakeGetStoreByIDMetrics{}

	uc := usecases.NewGetStoreByIDUseCase(
		repo,
		logger,
		metrics,
	)

	output, err := uc.Execute(context.Background(), storeID)

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

	require.True(t, repo.FindByIDCalled)
	require.Equal(t, storeID, repo.FindByIDValue)

	require.True(t, logger.successCalled)
	require.GreaterOrEqual(t, metrics.incrementCalls, 1)
	require.GreaterOrEqual(t, metrics.observeCalls, 1)
}

func TestGetStoreByIDUseCase_ReturnsNotFound(t *testing.T) {
	storeID := uuid.New()

	repo := &fixtures.MockStoreRepository{
		FindByIDErr: applicationerrors.ErrStoreNotFound,
	}
	logger := &fakeGetStoreByIDLogger{}
	metrics := &fakeGetStoreByIDMetrics{}

	uc := usecases.NewGetStoreByIDUseCase(
		repo,
		logger,
		metrics,
	)

	output, err := uc.Execute(context.Background(), storeID)

	require.ErrorIs(t, err, applicationerrors.ErrStoreNotFound)
	require.Equal(t, uuid.Nil, output.ID.Value())

	require.True(t, repo.FindByIDCalled)
	require.True(t, logger.failureCalled)
	require.GreaterOrEqual(t, metrics.incrementCalls, 1)
	require.GreaterOrEqual(t, metrics.observeCalls, 1)
}

func TestGetStoreByIDUseCase_PropagatesRepositoryFailure(t *testing.T) {
	storeID := uuid.New()
	repositoryError := errors.New("database unavailable")

	repo := &fixtures.MockStoreRepository{
		FindByIDErr: repositoryError,
	}
	logger := &fakeGetStoreByIDLogger{}
	metrics := &fakeGetStoreByIDMetrics{}

	uc := usecases.NewGetStoreByIDUseCase(
		repo,
		logger,
		metrics,
	)

	output, err := uc.Execute(context.Background(), storeID)

	require.ErrorIs(t, err, repositoryError)
	require.Equal(t, uuid.Nil, output.ID.Value())

	require.True(t, logger.failureCalled)
	require.GreaterOrEqual(t, metrics.incrementCalls, 1)
	require.GreaterOrEqual(t, metrics.observeCalls, 1)
}

func TestGetStoreByIDUseCase_LogsAndMeasuresSuccessfulRead(t *testing.T) {
	store, err := entities.NewStore(
		uuid.New(),
		"Observed Store",
		"observed-store",
		"Observed store description",
		nil,
		string(valueobjects.PlanTypeBasic),
	)
	require.NoError(t, err)

	logger := &fakeGetStoreByIDLogger{}
	metrics := &fakeGetStoreByIDMetrics{}

	uc := usecases.NewGetStoreByIDUseCase(
		&fixtures.MockStoreRepository{Store: store},
		logger,
		metrics,
	)

	_, err = uc.Execute(context.Background(), store.ID().Value())

	require.NoError(t, err)
	require.True(t, logger.successCalled)
	require.GreaterOrEqual(t, metrics.incrementCalls, 1)
	require.GreaterOrEqual(t, metrics.observeCalls, 1)
}

type fakeGetStoreByIDLogger struct {
	successCalled bool
	failureCalled bool
}

func (f *fakeGetStoreByIDLogger) Log(
	_ context.Context,
	event ports.LogEvent,
) error {
	switch event.Event {
	case "store.get_by_id.succeeded":
		f.successCalled = true
	case "store.get_by_id.failed":
		f.failureCalled = true
	}

	return nil
}

type fakeGetStoreByIDMetrics struct {
	incrementCalls int
	observeCalls   int
}

func (f *fakeGetStoreByIDMetrics) Increment(
	_ context.Context,
	_ ports.Metric,
) error {
	f.incrementCalls++
	return nil
}

func (f *fakeGetStoreByIDMetrics) Observe(
	_ context.Context,
	_ ports.Metric,
) error {
	f.observeCalls++
	return nil
}

func testTimestamp() time.Time {
	return time.Date(
		2026,
		9,
		19,
		12,
		0,
		0,
		0,
		time.UTC,
	)
}
