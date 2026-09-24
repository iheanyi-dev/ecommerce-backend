package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	usecases "github.com/iheanyi-dev/ecommerce-backend/services/store/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/test/unit/fixtures"
)

func TestCreateStoreUseCase_RequiresAuthentication(t *testing.T) {
	repo := &MockStoreRepository{}
	identity := &fakeIdentityProvider{}
	resolver := &fakeImageFormatResolver{}
	imageDispatcher := &fakeImageStorageDispatcher{}
	embeddingDispatcher := &fakeEmbeddingDispatcher{}
	logger := &fakeLogger{}
	metrics := &fakeMetrics{}

	uc := usecases.NewCreateStoreUseCase(
		repo,
		identity,
		resolver,
		imageDispatcher,
		embeddingDispatcher,
		logger,
		metrics,
	)

	_, err := uc.Execute(context.Background(), dto.CreateStoreInput{
		Name:        "Test Store",
		Slug:        "test-store",
		Description: "Test store description",
	})

	require.ErrorIs(t, err, applicationerrors.ErrUnauthenticated)
	require.False(t, repo.CreateCalled)
	require.False(t, identity.IsVendorCalled)
}

func TestCreateStoreUseCase_CreatesStoreAndPromotesOwnerToVendor(t *testing.T) {
	ownerID := uuid.New()

	repo := &MockStoreRepository{}
	identity := &fakeIdentityProvider{
		Vendor: false,
	}
	resolver := &fakeImageFormatResolver{}
	imageDispatcher := &fakeImageStorageDispatcher{}
	embeddingDispatcher := &fakeEmbeddingDispatcher{}
	logger := &fakeLogger{}
	metrics := &fakeMetrics{}

	uc := usecases.NewCreateStoreUseCase(
		repo,
		identity,
		resolver,
		imageDispatcher,
		embeddingDispatcher,
		logger,
		metrics,
	)

	ctx := ports.WithAuthenticatedIdentity(
		context.Background(),
		ports.AuthenticatedIdentity{
			UserID: ownerID,
			Role:   "user",
		},
	)

	output, err := uc.Execute(ctx, dto.CreateStoreInput{
		Name:        "Test Store",
		Slug:        "test-store",
		Description: "Test store description",
	})

	require.NoError(t, err)
	require.NotNil(t, output)

	require.True(t, repo.CreateCalled)
	require.NotNil(t, repo.CreatedStore)
	require.Equal(t, ownerID, repo.CreatedStore.OwnerID().Value())

	require.True(t, identity.IsVendorCalled)
	require.True(t, identity.PromoteCalled)
	require.Equal(t, ownerID, identity.PromotedUserID)

	require.True(t, embeddingDispatcher.Called)
	require.Equal(
		t,
		repo.CreatedStore.ID().Value(),
		embeddingDispatcher.Embedding.StoreID,
	)

	require.True(t, logger.SuccessCalled)
	require.GreaterOrEqual(t, metrics.IncrementCalls, 1)
}

func TestCreateStoreUseCase_GeneratesImageReferenceFromStoreIDAndContentType(t *testing.T) {
	ownerID := uuid.New()

	repo := &MockStoreRepository{}
	identity := &fakeIdentityProvider{}
	resolver := &fakeImageFormatResolver{
		Extension: "png",
	}
	imageDispatcher := &fakeImageStorageDispatcher{}
	embeddingDispatcher := &fakeEmbeddingDispatcher{}
	logger := &fakeLogger{}
	metrics := &fakeMetrics{}

	uc := usecases.NewCreateStoreUseCase(
		repo,
		identity,
		resolver,
		imageDispatcher,
		embeddingDispatcher,
		logger,
		metrics,
	)

	ctx := authenticatedContext(ownerID)

	output, err := uc.Execute(ctx, dto.CreateStoreInput{
		Name:        "Image Store",
		Slug:        "image-store",
		Description: "Store with image",
		Image: &dto.StoreImageInput{
			Content:     []byte("image-bytes"),
			Filename:    "logo.anything",
			ContentType: "image/png",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	require.NotNil(t, repo.CreatedStore.ImageReference())

	expectedReference := "stores/" +
		repo.CreatedStore.ID().Value().String() +
		".png"

	require.Equal(
		t,
		expectedReference,
		*repo.CreatedStore.ImageReference(),
	)

	require.True(t, resolver.Called)
	require.Equal(t, "image/png", resolver.ContentType)

	require.True(t, imageDispatcher.Called)
	require.Equal(
		t,
		repo.CreatedStore.ID().Value(),
		imageDispatcher.Image.StoreID,
	)
	require.Equal(t, expectedReference, imageDispatcher.Image.Reference)
	require.Equal(t, []byte("image-bytes"), imageDispatcher.Image.Content)
	require.Equal(t, "image/png", imageDispatcher.Image.ContentType)
}

func TestCreateStoreUseCase_OwnerAlreadyExists_ReconcilesIdentityAndDispatchesExistingStore(t *testing.T) {
	ownerID := uuid.New()

	existingStore, err := entities.NewStore(
		ownerID,
		"Existing Store",
		"existing-store",
		"Existing store description",
		nil,
		string(valueobjects.PlanTypeBasic),
	)
	require.NoError(t, err)

	repo := &fixtures.MockStoreRepository{
		CreateError: applicationerrors.ErrStoreAlreadyExists,
		OwnerStore:  existingStore,
	}
	identity := &fakeIdentityProvider{
		Vendor: false,
	}
	resolver := &fakeImageFormatResolver{}
	imageDispatcher := &fakeImageStorageDispatcher{}
	embeddingDispatcher := &fakeEmbeddingDispatcher{}
	logger := &fakeLogger{}
	metrics := &fakeMetrics{}

	uc := usecases.NewCreateStoreUseCase(
		repo,
		identity,
		resolver,
		imageDispatcher,
		embeddingDispatcher,
		logger,
		metrics,
	)

	output, err := uc.Execute(
		authenticatedContext(ownerID),
		dto.CreateStoreInput{
			Name:        "New Store Attempt",
			Slug:        "new-store-attempt",
			Description: "New store attempt",
		},
	)

	require.NoError(t, err)
	require.NotNil(t, output)
	require.Equal(t, existingStore.ID().Value(), output.ID.Value())

	require.True(t, repo.FindByOwnerCalled)
	require.Equal(t, ownerID, repo.FindByOwnerID)

	require.True(t, identity.PromoteCalled)
	require.Equal(t, ownerID, identity.PromotedUserID)

	require.True(t, embeddingDispatcher.Called)
	require.Equal(
		t,
		existingStore.ID().Value(),
		embeddingDispatcher.Embedding.StoreID,
	)
}

func TestCreateStoreUseCase_OwnerAlreadyExists_IdentityPromotionFailure_DeletesExistingStore(t *testing.T) {
	ownerID := uuid.New()

	existingStore, err := entities.NewStore(
		ownerID,
		"Existing Store",
		"existing-store",
		"Existing store description",
		nil,
		string(valueobjects.PlanTypeBasic),
	)
	require.NoError(t, err)

	promotionError := errors.New("identity promotion failed")

	repo := &fixtures.MockStoreRepository{
		CreateError: applicationerrors.ErrStoreAlreadyExists,
		OwnerStore:  existingStore,
	}
	identity := &fakeIdentityProvider{
		Vendor:       false,
		PromoteError: promotionError,
	}
	logger := &fakeLogger{}
	metrics := &fakeMetrics{}

	uc := usecases.NewCreateStoreUseCase(
		repo,
		identity,
		&fakeImageFormatResolver{},
		&fakeImageStorageDispatcher{},
		&fakeEmbeddingDispatcher{},
		logger,
		metrics,
	)

	_, err = uc.Execute(
		authenticatedContext(ownerID),
		dto.CreateStoreInput{
			Name:        "New Store Attempt",
			Slug:        "new-store-attempt",
			Description: "New store attempt",
		},
	)

	require.Error(t, err)
	require.True(t, repo.DeleteCalled)
	require.Equal(t, existingStore.ID().Value(), repo.DeletedStoreID)
}

func TestCreateStoreUseCase_IdentityPromotionFailure_DeletesPersistedStore(t *testing.T) {
	ownerID := uuid.New()
	promotionError := errors.New("identity promotion failed")

	repo := &MockStoreRepository{}
	identity := &fakeIdentityProvider{
		Vendor:       false,
		PromoteError: promotionError,
	}
	logger := &fakeLogger{}
	metrics := &fakeMetrics{}

	uc := usecases.NewCreateStoreUseCase(
		repo,
		identity,
		&fakeImageFormatResolver{},
		&fakeImageStorageDispatcher{},
		&fakeEmbeddingDispatcher{},
		logger,
		metrics,
	)

	_, err := uc.Execute(
		authenticatedContext(ownerID),
		dto.CreateStoreInput{
			Name:        "Compensation Store",
			Slug:        "compensation-store",
			Description: "Compensation test",
		},
	)

	require.Error(t, err)
	require.True(t, repo.DeleteCalled)
	require.Equal(
		t,
		repo.CreatedStore.ID().Value(),
		repo.DeletedStoreID,
	)
}

func TestCreateStoreUseCase_IdentityPromotionFailureAndDeleteFailure_ReturnsError(t *testing.T) {
	ownerID := uuid.New()

	repo := &fixtures.MockStoreRepository{
		DeleteError: errors.New("delete failed"),
	}
	identity := &fakeIdentityProvider{
		Vendor:       false,
		PromoteError: errors.New("identity promotion failed"),
	}
	logger := &fakeLogger{}
	metrics := &fakeMetrics{}

	uc := usecases.NewCreateStoreUseCase(
		repo,
		identity,
		&fakeImageFormatResolver{},
		&fakeImageStorageDispatcher{},
		&fakeEmbeddingDispatcher{},
		logger,
		metrics,
	)

	_, err := uc.Execute(
		authenticatedContext(ownerID),
		dto.CreateStoreInput{
			Name:        "Compensation Failure Store",
			Slug:        "compensation-failure-store",
			Description: "Compensation failure test",
		},
	)

	require.Error(t, err)
	require.True(t, repo.DeleteCalled)
}

func TestCreateStoreUseCase_AsynchronousDispatchFailuresDoNotFailStoreCreation(t *testing.T) {
	ownerID := uuid.New()

	repo := &MockStoreRepository{}
	identity := &fakeIdentityProvider{
		Vendor: false,
	}
	imageDispatcher := &fakeImageStorageDispatcher{
		DispatchError: errors.New("image dispatch failed"),
	}
	embeddingDispatcher := &fakeEmbeddingDispatcher{
		DispatchError: errors.New("embedding dispatch failed"),
	}
	logger := &fakeLogger{}
	metrics := &fakeMetrics{}

	uc := usecases.NewCreateStoreUseCase(
		repo,
		identity,
		&fakeImageFormatResolver{
			Extension: "webp",
		},
		imageDispatcher,
		embeddingDispatcher,
		logger,
		metrics,
	)

	output, err := uc.Execute(
		authenticatedContext(ownerID),
		dto.CreateStoreInput{
			Name:        "Async Store",
			Slug:        "async-store",
			Description: "Async dispatch failure test",
			Image: &dto.StoreImageInput{
				Content:     []byte("image"),
				Filename:    "logo",
				ContentType: "image/webp",
			},
		},
	)

	require.NoError(t, err)
	require.NotNil(t, output)
	require.True(t, imageDispatcher.Called)
	require.True(t, embeddingDispatcher.Called)
	require.True(t, logger.ImageDispatchFailureCalled)
	require.True(t, logger.EmbeddingDispatchFailureCalled)
}

func TestCreateStoreUseCase_ObservesSuccessfulCreation(t *testing.T) {
	ownerID := uuid.New()

	logger := &fakeLogger{}
	metrics := &fakeMetrics{}

	uc := usecases.NewCreateStoreUseCase(
		&MockStoreRepository{},
		&fakeIdentityProvider{},
		&fakeImageFormatResolver{},
		&fakeImageStorageDispatcher{},
		&fakeEmbeddingDispatcher{},
		logger,
		metrics,
	)

	_, err := uc.Execute(
		authenticatedContext(ownerID),
		dto.CreateStoreInput{
			Name:        "Observed Store",
			Slug:        "observed-store",
			Description: "Observed store",
		},
	)

	require.NoError(t, err)
	require.True(t, logger.SuccessCalled)
	require.GreaterOrEqual(t, metrics.IncrementCalls, 1)
	require.GreaterOrEqual(t, metrics.ObserveCalls, 1)
}

func TestCreateStoreUseCase_IdentityCheckFailureDoesNotPersistStore(t *testing.T) {
	ownerID := uuid.New()

	repo := &MockStoreRepository{}
	identity := &fakeIdentityProvider{
		IsVendorError: errors.New("identity unavailable"),
	}
	logger := &fakeLogger{}
	metrics := &fakeMetrics{}

	uc := usecases.NewCreateStoreUseCase(
		repo,
		identity,
		&fakeImageFormatResolver{},
		&fakeImageStorageDispatcher{},
		&fakeEmbeddingDispatcher{},
		logger,
		metrics,
	)

	_, err := uc.Execute(
		authenticatedContext(ownerID),
		dto.CreateStoreInput{
			Name:        "Identity Failure Store",
			Slug:        "identity-failure-store",
			Description: "Identity failure test",
		},
	)

	require.Error(t, err)
	require.False(t, repo.CreateCalled)
}

func authenticatedContext(userID uuid.UUID) context.Context {
	return ports.WithAuthenticatedIdentity(
		context.Background(),
		ports.AuthenticatedIdentity{
			UserID: userID,
			Role:   "user",
		},
	)
}

type fakeIdentityProvider struct {
	Vendor         bool
	IsVendorCalled bool
	IsVendorError  error
	PromoteCalled  bool
	PromotedUserID uuid.UUID
	PromoteError   error
}

func (f *fakeIdentityProvider) IsVendor(
	_ context.Context,
	_ uuid.UUID,
) (bool, error) {
	f.IsVendorCalled = true

	return f.Vendor, f.IsVendorError
}

func (f *fakeIdentityProvider) PromoteToVendor(
	_ context.Context,
	userID uuid.UUID,
) error {
	f.PromoteCalled = true
	f.PromotedUserID = userID

	return f.PromoteError
}

type fakeImageFormatResolver struct {
	Called      bool
	ContentType string
	Extension   string
	Err         error
}

func (f *fakeImageFormatResolver) ResolveExtension(
	_ context.Context,
	contentType string,
) (string, error) {
	f.Called = true
	f.ContentType = contentType

	return f.Extension, f.Err
}

type fakeImageStorageDispatcher struct {
	Called        bool
	Image         ports.StoreImage
	DispatchError error
}

func (f *fakeImageStorageDispatcher) DispatchStoreImage(
	_ context.Context,
	image ports.StoreImage,
) error {
	f.Called = true
	f.Image = image

	return f.DispatchError
}

type fakeEmbeddingDispatcher struct {
	Called        bool
	Embedding     ports.StoreEmbedding
	DispatchError error
}

func (f *fakeEmbeddingDispatcher) DispatchStoreEmbedding(
	_ context.Context,
	embedding ports.StoreEmbedding,
) error {
	f.Called = true
	f.Embedding = embedding

	return f.DispatchError
}

type fakeLogger struct {
	SuccessCalled                  bool
	ImageDispatchFailureCalled     bool
	EmbeddingDispatchFailureCalled bool
}

func (f *fakeLogger) Log(
	_ context.Context,
	event ports.LogEvent,
) error {
	switch event.Event {
	case "store.create.succeeded":
		f.SuccessCalled = true

	case "store.create.image_dispatch_failed":
		f.ImageDispatchFailureCalled = true

	case "store.create.embedding_dispatch_failed":
		f.EmbeddingDispatchFailureCalled = true
	}

	return nil
}

type fakeMetrics struct {
	IncrementCalls int
	ObserveCalls   int
}

func (f *fakeMetrics) Increment(
	_ context.Context,
	_ ports.Metric,
) error {
	f.IncrementCalls++

	return nil
}

func (f *fakeMetrics) Observe(
	_ context.Context,
	_ ports.Metric,
) error {
	f.ObserveCalls++

	return nil
}

type MockStoreRepository struct {
	CreateCalled      bool
	CreatedStore      *entities.Store
	CreateError       error
	OwnerStore        *entities.Store
	FindByOwnerCalled bool
	OwnerID           uuid.UUID

	DeleteCalled   bool
	DeletedStoreID uuid.UUID
	DeleteError    error

	ChangePlanCalled  bool
	ChangePlanStoreID uuid.UUID
	ChangedPlanStore  *entities.Store
	ChangePlanErr     error
	Store             *entities.Store

	FindByIDCalled bool
	FindByIDErr    error

	ChangeStatusCalled  bool
	ChangeStatusStoreID uuid.UUID
	ChangeStatusValue   string
	ChangeStatusErr     error
}

func (f *MockStoreRepository) Create(
	_ context.Context,
	store *entities.Store,
) error {
	f.CreateCalled = true
	f.CreatedStore = store

	return f.CreateError
}

func (f *MockStoreRepository) FindByID(
	_ context.Context,
	storeID uuid.UUID,
) (*entities.Store, error) {
	f.FindByIDCalled = true

	// Keep the requested ID available through the mock's public state.
	f.Store = nil

	_ = storeID

	return nil, f.FindByIDErr
}

func (f *MockStoreRepository) FindByOwnerID(
	_ context.Context,
	ownerID uuid.UUID,
) (*entities.Store, error) {
	f.FindByOwnerCalled = true
	f.OwnerID = ownerID

	return f.OwnerStore, nil
}

func (f *MockStoreRepository) FindBySlug(
	_ context.Context,
	_ string,
) (*entities.Store, error) {
	return nil, nil
}

func (f *MockStoreRepository) ExistsBySlug(
	_ context.Context,
	_ string,
) (bool, error) {
	return true, nil
}

func (f *MockStoreRepository) ListActive(
	_ context.Context,
	_ string,
	_ int,
	_ int,
) ([]*entities.Store, int, error) {
	return nil, 0, nil
}

func (f *MockStoreRepository) Update(
	_ context.Context,
	_ *entities.Store,
) error {
	return nil
}

func (f *MockStoreRepository) Delete(
	_ context.Context,
	storeID uuid.UUID,
) error {
	f.DeleteCalled = true
	f.DeletedStoreID = storeID

	return f.DeleteError
}

func (f *MockStoreRepository) ChangePlan(
	_ context.Context,
	store *entities.Store,
) error {
	return nil
}

func (f *MockStoreRepository) ChangeStatus(
	_ context.Context,
	storeID uuid.UUID,
	status string,
) error {
	return nil
}
