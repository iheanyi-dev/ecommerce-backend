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
)

func TestCreateStoreUseCase_RequiresAuthentication(t *testing.T) {
	repo := &fakeStoreRepository{}
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
	require.False(t, repo.createCalled)
	require.False(t, identity.isVendorCalled)
}

func TestCreateStoreUseCase_CreatesStoreAndPromotesOwnerToVendor(t *testing.T) {
	ownerID := uuid.New()

	repo := &fakeStoreRepository{}
	identity := &fakeIdentityProvider{
		isVendor: false,
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

	require.True(t, repo.createCalled)
	require.NotNil(t, repo.createdStore)
	require.Equal(t, ownerID, repo.createdStore.OwnerID().Value())

	require.True(t, identity.isVendorCalled)
	require.True(t, identity.promoteCalled)
	require.Equal(t, ownerID, identity.promotedUserID)

	require.True(t, embeddingDispatcher.called)
	require.Equal(t, repo.createdStore.ID().Value(), embeddingDispatcher.embedding.StoreID)

	require.True(t, logger.successCalled)
	require.GreaterOrEqual(t, metrics.incrementCalls, 1)
}

func TestCreateStoreUseCase_GeneratesImageReferenceFromStoreIDAndContentType(t *testing.T) {
	ownerID := uuid.New()

	repo := &fakeStoreRepository{}
	identity := &fakeIdentityProvider{}
	resolver := &fakeImageFormatResolver{
		extension: "png",
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
	require.NotNil(t, repo.createdStore.ImageReference())

	expectedReference := "stores/" + repo.createdStore.ID().Value().String() + ".png"
	require.Equal(t, expectedReference, *repo.createdStore.ImageReference())

	require.True(t, resolver.called)
	require.Equal(t, "image/png", resolver.contentType)

	require.True(t, imageDispatcher.called)
	require.Equal(t, repo.createdStore.ID().Value(), imageDispatcher.image.StoreID)
	require.Equal(t, expectedReference, imageDispatcher.image.Reference)
	require.Equal(t, []byte("image-bytes"), imageDispatcher.image.Content)
	require.Equal(t, "image/png", imageDispatcher.image.ContentType)
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

	repo := &fakeStoreRepository{
		createError: applicationerrors.ErrStoreAlreadyExists,
		ownerStore:  existingStore,
	}
	identity := &fakeIdentityProvider{
		isVendor: false,
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

	require.True(t, repo.findByOwnerCalled)
	require.Equal(t, ownerID, repo.findByOwnerID)

	require.True(t, identity.promoteCalled)
	require.Equal(t, ownerID, identity.promotedUserID)

	require.True(t, embeddingDispatcher.called)
	require.Equal(t, existingStore.ID().Value(), embeddingDispatcher.embedding.StoreID)
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

	repo := &fakeStoreRepository{
		createError: applicationerrors.ErrStoreAlreadyExists,
		ownerStore:  existingStore,
	}
	identity := &fakeIdentityProvider{
		isVendor:     false,
		promoteError: promotionError,
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
	require.True(t, repo.deleteCalled)
	require.Equal(t, existingStore.ID().Value(), repo.deletedStoreID)
}

func TestCreateStoreUseCase_IdentityPromotionFailure_DeletesPersistedStore(t *testing.T) {
	ownerID := uuid.New()
	promotionError := errors.New("identity promotion failed")

	repo := &fakeStoreRepository{}
	identity := &fakeIdentityProvider{
		isVendor:     false,
		promoteError: promotionError,
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
	require.True(t, repo.deleteCalled)
	require.Equal(t, repo.createdStore.ID().Value(), repo.deletedStoreID)
}

func TestCreateStoreUseCase_IdentityPromotionFailureAndDeleteFailure_ReturnsError(t *testing.T) {
	ownerID := uuid.New()

	repo := &fakeStoreRepository{
		deleteError: errors.New("delete failed"),
	}
	identity := &fakeIdentityProvider{
		isVendor:     false,
		promoteError: errors.New("identity promotion failed"),
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
	require.True(t, repo.deleteCalled)
}

func TestCreateStoreUseCase_AsynchronousDispatchFailuresDoNotFailStoreCreation(t *testing.T) {
	ownerID := uuid.New()

	repo := &fakeStoreRepository{}
	identity := &fakeIdentityProvider{
		isVendor: false,
	}
	imageDispatcher := &fakeImageStorageDispatcher{
		dispatchError: errors.New("image dispatch failed"),
	}
	embeddingDispatcher := &fakeEmbeddingDispatcher{
		dispatchError: errors.New("embedding dispatch failed"),
	}
	logger := &fakeLogger{}
	metrics := &fakeMetrics{}

	uc := usecases.NewCreateStoreUseCase(
		repo,
		identity,
		&fakeImageFormatResolver{
			extension: "webp",
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
	require.True(t, imageDispatcher.called)
	require.True(t, embeddingDispatcher.called)
	require.True(t, logger.imageDispatchFailureCalled)
	require.True(t, logger.embeddingDispatchFailureCalled)
}

func TestCreateStoreUseCase_ObservesSuccessfulCreation(t *testing.T) {
	ownerID := uuid.New()

	logger := &fakeLogger{}
	metrics := &fakeMetrics{}

	uc := usecases.NewCreateStoreUseCase(
		&fakeStoreRepository{},
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
	require.True(t, logger.successCalled)
	require.GreaterOrEqual(t, metrics.incrementCalls, 1)
	require.GreaterOrEqual(t, metrics.observeCalls, 1)
}

func TestCreateStoreUseCase_IdentityCheckFailureDoesNotPersistStore(t *testing.T) {
	ownerID := uuid.New()

	repo := &fakeStoreRepository{}
	identity := &fakeIdentityProvider{
		isVendorError: errors.New("identity unavailable"),
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
	require.False(t, repo.createCalled)
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

type fakeStoreRepository struct {
	createCalled      bool
	createdStore      *entities.Store
	createError       error
	ownerStore        *entities.Store
	findByOwnerCalled bool
	findByOwnerID     uuid.UUID
	deleteCalled      bool
	deletedStoreID    uuid.UUID
	deleteError       error
}

func (f *fakeStoreRepository) Create(_ context.Context, store *entities.Store) error {
	f.createCalled = true
	f.createdStore = store
	return f.createError
}

func (f *fakeStoreRepository) FindByID(_ context.Context, _ uuid.UUID) (*entities.Store, error) {
	return nil, nil
}

func (f *fakeStoreRepository) FindByOwnerID(_ context.Context, ownerID uuid.UUID) (*entities.Store, error) {
	f.findByOwnerCalled = true
	f.findByOwnerID = ownerID
	return f.ownerStore, nil
}

func (f *fakeStoreRepository) FindBySlug(_ context.Context, _ string) (*entities.Store, error) {
	return nil, nil
}

func (f *fakeStoreRepository) ExistsBySlug(_ context.Context, _ string) (bool, error) {
	return true, nil
}

func (f *fakeStoreRepository) ListActive(
	ctx context.Context,
	query string,
	page int,
	pageSize int,
) ([]*entities.Store, int, error) {
	return nil, 0, nil
}

func (f *fakeStoreRepository) Update(_ context.Context, _ *entities.Store) error {
	return nil
}

func (f *fakeStoreRepository) Delete(_ context.Context, storeID uuid.UUID) error {
	f.deleteCalled = true
	f.deletedStoreID = storeID
	return f.deleteError
}

type fakeIdentityProvider struct {
	isVendor       bool
	isVendorCalled bool
	isVendorError  error
	promoteCalled  bool
	promotedUserID uuid.UUID
	promoteError   error
}

func (f *fakeIdentityProvider) IsVendor(_ context.Context, _ uuid.UUID) (bool, error) {
	f.isVendorCalled = true
	return f.isVendor, f.isVendorError
}

func (f *fakeIdentityProvider) PromoteToVendor(_ context.Context, userID uuid.UUID) error {
	f.promoteCalled = true
	f.promotedUserID = userID
	return f.promoteError
}

type fakeImageFormatResolver struct {
	called      bool
	contentType string
	extension   string
	err         error
}

func (f *fakeImageFormatResolver) ResolveExtension(
	_ context.Context,
	contentType string,
) (string, error) {
	f.called = true
	f.contentType = contentType
	return f.extension, f.err
}

type fakeImageStorageDispatcher struct {
	called        bool
	image         ports.StoreImage
	dispatchError error
}

func (f *fakeImageStorageDispatcher) DispatchStoreImage(
	_ context.Context,
	image ports.StoreImage,
) error {
	f.called = true
	f.image = image
	return f.dispatchError
}

type fakeEmbeddingDispatcher struct {
	called        bool
	embedding     ports.StoreEmbedding
	dispatchError error
}

func (f *fakeEmbeddingDispatcher) DispatchStoreEmbedding(
	_ context.Context,
	embedding ports.StoreEmbedding,
) error {
	f.called = true
	f.embedding = embedding
	return f.dispatchError
}

type fakeLogger struct {
	successCalled                  bool
	imageDispatchFailureCalled     bool
	embeddingDispatchFailureCalled bool
}

func (f *fakeLogger) Log(_ context.Context, event ports.LogEvent) error {
	switch event.Event {
	case "store.create.succeeded":
		f.successCalled = true
	case "store.create.image_dispatch_failed":
		f.imageDispatchFailureCalled = true
	case "store.create.embedding_dispatch_failed":
		f.embeddingDispatchFailureCalled = true
	}

	return nil
}

type fakeMetrics struct {
	incrementCalls int
	observeCalls   int
}

func (f *fakeMetrics) Increment(_ context.Context, _ ports.Metric) error {
	f.incrementCalls++
	return nil
}

func (f *fakeMetrics) Observe(_ context.Context, _ ports.Metric) error {
	f.observeCalls++
	return nil
}
