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

func TestChangeStoreImageUseCase_RequiresAuthentication(t *testing.T) {
	store := newChangeImageTestStore(t)

	repo := &changeStoreImageRepository{
		store: store,
	}
	resolver := &changeStoreImageFormatResolver{
		extension: "png",
	}
	imageDispatcher := &changeStoreImageDispatcher{}
	logger := &changeStoreImageLogger{}
	metrics := &changeStoreImageMetrics{}

	uc := usecases.NewChangeStoreImageUseCase(
		repo,
		resolver,
		imageDispatcher,
		logger,
		metrics,
	)

	_, err := uc.Execute(
		context.Background(),
		dto.ChangeStoreImageInput{
			StoreID: store.ID().Value(),
			Image: &dto.StoreImageInput{
				Content:     []byte("image"),
				Filename:    "logo.png",
				ContentType: "image/png",
			},
		},
	)

	require.ErrorIs(t, err, applicationerrors.ErrUnauthenticated)
	require.False(t, repo.findByIDCalled)
	require.False(t, imageDispatcher.called)
	require.False(t, repo.updateCalled)
}

func TestChangeStoreImageUseCase_StoreNotFound(t *testing.T) {
	ownerID := uuid.New()
	storeID := uuid.New()

	repo := &changeStoreImageRepository{
		findByIDResult: nil,
	}
	resolver := &changeStoreImageFormatResolver{
		extension: "png",
	}
	imageDispatcher := &changeStoreImageDispatcher{}
	logger := &changeStoreImageLogger{}
	metrics := &changeStoreImageMetrics{}

	uc := usecases.NewChangeStoreImageUseCase(
		repo,
		resolver,
		imageDispatcher,
		logger,
		metrics,
	)

	_, err := uc.Execute(
		authenticatedChangeImageContext(ownerID),
		dto.ChangeStoreImageInput{
			StoreID: storeID,
			Image: &dto.StoreImageInput{
				Content:     []byte("image"),
				Filename:    "logo.png",
				ContentType: "image/png",
			},
		},
	)

	require.ErrorIs(t, err, applicationerrors.ErrStoreNotFound)
	require.True(t, repo.findByIDCalled)
	require.False(t, imageDispatcher.called)
	require.False(t, repo.updateCalled)
}

func TestChangeStoreImageUseCase_RepositoryLookupFailure(t *testing.T) {
	ownerID := uuid.New()
	storeID := uuid.New()
	lookupError := errors.New("database unavailable")

	repo := &changeStoreImageRepository{
		findByIDError: lookupError,
	}
	resolver := &changeStoreImageFormatResolver{
		extension: "png",
	}
	imageDispatcher := &changeStoreImageDispatcher{}
	logger := &changeStoreImageLogger{}
	metrics := &changeStoreImageMetrics{}

	uc := usecases.NewChangeStoreImageUseCase(
		repo,
		resolver,
		imageDispatcher,
		logger,
		metrics,
	)

	_, err := uc.Execute(
		authenticatedChangeImageContext(ownerID),
		dto.ChangeStoreImageInput{
			StoreID: storeID,
			Image: &dto.StoreImageInput{
				Content:     []byte("image"),
				Filename:    "logo.png",
				ContentType: "image/png",
			},
		},
	)

	require.Error(t, err)
	require.ErrorIs(t, err, lookupError)
	require.False(t, imageDispatcher.called)
	require.False(t, repo.updateCalled)
}

func TestChangeStoreImageUseCase_NonOwnerReceivesStoreNotFound(t *testing.T) {
	ownerID := uuid.New()
	otherUserID := uuid.New()

	store := newChangeImageTestStoreWithOwner(t, ownerID)

	repo := &changeStoreImageRepository{
		store: store,
	}
	resolver := &changeStoreImageFormatResolver{
		extension: "png",
	}
	imageDispatcher := &changeStoreImageDispatcher{}
	logger := &changeStoreImageLogger{}
	metrics := &changeStoreImageMetrics{}

	uc := usecases.NewChangeStoreImageUseCase(
		repo,
		resolver,
		imageDispatcher,
		logger,
		metrics,
	)

	_, err := uc.Execute(
		authenticatedChangeImageContext(otherUserID),
		dto.ChangeStoreImageInput{
			StoreID: store.ID().Value(),
			Image: &dto.StoreImageInput{
				Content:     []byte("image"),
				Filename:    "logo.png",
				ContentType: "image/png",
			},
		},
	)

	require.ErrorIs(t, err, applicationerrors.ErrStoreNotFound)
	require.False(t, resolver.called)
	require.False(t, imageDispatcher.called)
	require.False(t, repo.updateCalled)
}

func TestChangeStoreImageUseCase_StoresImageBeforePersistingNewReference(t *testing.T) {
	ownerID := uuid.New()
	store := newChangeImageTestStoreWithOwner(t, ownerID)

	repo := &changeStoreImageRepository{
		store: store,
	}
	resolver := &changeStoreImageFormatResolver{
		extension: "png",
	}
	imageDispatcher := &changeStoreImageDispatcher{}
	logger := &changeStoreImageLogger{}
	metrics := &changeStoreImageMetrics{}

	uc := usecases.NewChangeStoreImageUseCase(
		repo,
		resolver,
		imageDispatcher,
		logger,
		metrics,
	)

	originalReference := "stores/" + store.ID().Value().String() + ".jpg"
	store.SetImageReference(originalReference)

	output, err := uc.Execute(
		authenticatedChangeImageContext(ownerID),
		dto.ChangeStoreImageInput{
			StoreID: store.ID().Value(),
			Image: &dto.StoreImageInput{
				Content:     []byte("new-image-bytes"),
				Filename:    "logo.anything",
				ContentType: "image/png",
			},
		},
	)

	require.NoError(t, err)
	require.NotNil(t, output)

	expectedReference := "stores/" + store.ID().Value().String() + ".png"

	require.True(t, resolver.called)
	require.Equal(t, "image/png", resolver.contentType)

	require.True(t, imageDispatcher.called)
	require.Equal(t, store.ID().Value(), imageDispatcher.image.StoreID)
	require.Equal(t, expectedReference, imageDispatcher.image.Reference)
	require.Equal(t, []byte("new-image-bytes"), imageDispatcher.image.Content)
	require.Equal(t, "image/png", imageDispatcher.image.ContentType)

	require.True(t, repo.updateCalled)
	require.Same(t, store, repo.updatedStore)
	require.Equal(t, expectedReference, *repo.updatedStore.ImageReference())

	require.Equal(t, expectedReference, *output.ImageReference)

	require.True(t, imageDispatcher.calledBeforeRepositoryUpdate)
}

func TestChangeStoreImageUseCase_StorageFailureDoesNotChangeOrPersistImageReference(t *testing.T) {
	ownerID := uuid.New()
	store := newChangeImageTestStoreWithOwner(t, ownerID)

	originalReference := "stores/" + store.ID().Value().String() + ".jpg"
	store.SetImageReference(originalReference)

	storageError := errors.New("storage service unavailable")

	repo := &changeStoreImageRepository{
		store: store,
	}
	resolver := &changeStoreImageFormatResolver{
		extension: "png",
	}
	imageDispatcher := &changeStoreImageDispatcher{
		dispatchError: storageError,
	}
	logger := &changeStoreImageLogger{}
	metrics := &changeStoreImageMetrics{}

	uc := usecases.NewChangeStoreImageUseCase(
		repo,
		resolver,
		imageDispatcher,
		logger,
		metrics,
	)

	_, err := uc.Execute(
		authenticatedChangeImageContext(ownerID),
		dto.ChangeStoreImageInput{
			StoreID: store.ID().Value(),
			Image: &dto.StoreImageInput{
				Content:     []byte("new-image"),
				Filename:    "logo.png",
				ContentType: "image/png",
			},
		},
	)

	require.Error(t, err)
	require.ErrorIs(t, err, storageError)

	require.NotNil(t, store.ImageReference())
	require.Equal(t, originalReference, *store.ImageReference())

	require.False(t, repo.updateCalled)
	require.True(t, imageDispatcher.called)
}

func TestChangeStoreImageUseCase_ImageFormatResolutionFailureDoesNotStoreOrPersist(t *testing.T) {
	ownerID := uuid.New()
	store := newChangeImageTestStoreWithOwner(t, ownerID)

	resolverError := errors.New("unsupported image format")

	repo := &changeStoreImageRepository{
		store: store,
	}
	resolver := &changeStoreImageFormatResolver{
		err: resolverError,
	}
	imageDispatcher := &changeStoreImageDispatcher{}
	logger := &changeStoreImageLogger{}
	metrics := &changeStoreImageMetrics{}

	uc := usecases.NewChangeStoreImageUseCase(
		repo,
		resolver,
		imageDispatcher,
		logger,
		metrics,
	)

	_, err := uc.Execute(
		authenticatedChangeImageContext(ownerID),
		dto.ChangeStoreImageInput{
			StoreID: store.ID().Value(),
			Image: &dto.StoreImageInput{
				Content:     []byte("image"),
				Filename:    "logo.unknown",
				ContentType: "image/unknown",
			},
		},
	)

	require.Error(t, err)
	require.ErrorIs(t, err, resolverError)

	require.False(t, imageDispatcher.called)
	require.False(t, repo.updateCalled)
	require.Nil(t, store.ImageReference())
}

func TestChangeStoreImageUseCase_PersistenceFailureOccursAfterSuccessfulStorage(t *testing.T) {
	ownerID := uuid.New()
	store := newChangeImageTestStoreWithOwner(t, ownerID)

	persistenceError := errors.New("database update failed")

	repo := &changeStoreImageRepository{
		store:       store,
		updateError: persistenceError,
	}
	resolver := &changeStoreImageFormatResolver{
		extension: "webp",
	}
	imageDispatcher := &changeStoreImageDispatcher{}
	logger := &changeStoreImageLogger{}
	metrics := &changeStoreImageMetrics{}

	uc := usecases.NewChangeStoreImageUseCase(
		repo,
		resolver,
		imageDispatcher,
		logger,
		metrics,
	)

	_, err := uc.Execute(
		authenticatedChangeImageContext(ownerID),
		dto.ChangeStoreImageInput{
			StoreID: store.ID().Value(),
			Image: &dto.StoreImageInput{
				Content:     []byte("image"),
				Filename:    "logo.webp",
				ContentType: "image/webp",
			},
		},
	)

	require.Error(t, err)
	require.ErrorIs(t, err, persistenceError)

	require.True(t, imageDispatcher.called)
	require.True(t, repo.updateCalled)

	expectedReference := "stores/" + store.ID().Value().String() + ".webp"
	require.Equal(t, expectedReference, *store.ImageReference())
}

func TestChangeStoreImageUseCase_ObservesSuccessfulChange(t *testing.T) {
	ownerID := uuid.New()
	store := newChangeImageTestStoreWithOwner(t, ownerID)

	repo := &changeStoreImageRepository{
		store: store,
	}
	resolver := &changeStoreImageFormatResolver{
		extension: "png",
	}
	imageDispatcher := &changeStoreImageDispatcher{}
	logger := &changeStoreImageLogger{}
	metrics := &changeStoreImageMetrics{}

	uc := usecases.NewChangeStoreImageUseCase(
		repo,
		resolver,
		imageDispatcher,
		logger,
		metrics,
	)

	_, err := uc.Execute(
		authenticatedChangeImageContext(ownerID),
		dto.ChangeStoreImageInput{
			StoreID: store.ID().Value(),
			Image: &dto.StoreImageInput{
				Content:     []byte("image"),
				Filename:    "logo.png",
				ContentType: "image/png",
			},
		},
	)

	require.NoError(t, err)
	require.True(t, logger.successCalled)
	require.GreaterOrEqual(t, metrics.incrementCalls, 1)
	require.GreaterOrEqual(t, metrics.observeCalls, 1)
}

func authenticatedChangeImageContext(userID uuid.UUID) context.Context {
	return ports.WithAuthenticatedIdentity(
		context.Background(),
		ports.AuthenticatedIdentity{
			UserID: userID,
			Role:   "user",
		},
	)
}

func newChangeImageTestStore(t *testing.T) *entities.Store {
	return newChangeImageTestStoreWithOwner(t, uuid.New())
}

func newChangeImageTestStoreWithOwner(
	t *testing.T,
	ownerID uuid.UUID,
) *entities.Store {
	store, err := entities.NewStore(
		ownerID,
		"Image Store",
		"image-store",
		"Image store description",
		nil,
		string(valueobjects.PlanTypeBasic),
	)
	require.NoError(t, err)

	return store
}

type changeStoreImageRepository struct {
	store          *entities.Store
	findByIDResult *entities.Store
	findByIDError  error
	findByIDCalled bool
	updateCalled   bool
	updatedStore   *entities.Store
	updateError    error
}

func (f *changeStoreImageRepository) Create(
	_ context.Context,
	_ *entities.Store,
) error {
	return nil
}

func (f *changeStoreImageRepository) Delete(
	_ context.Context,
	_ uuid.UUID,
) error {
	return nil
}

func (f *changeStoreImageRepository) FindByID(
	_ context.Context,
	storeID uuid.UUID,
) (*entities.Store, error) {
	f.findByIDCalled = true

	if f.findByIDError != nil {
		return nil, f.findByIDError
	}

	if f.findByIDResult != nil {
		return f.findByIDResult, nil
	}

	if f.store != nil && f.store.ID().Value() == storeID {
		return f.store, nil
	}

	return nil, nil
}

func (f *changeStoreImageRepository) FindByOwnerID(
	_ context.Context,
	_ uuid.UUID,
) (*entities.Store, error) {
	return nil, nil
}

func (f *changeStoreImageRepository) FindBySlug(
	_ context.Context,
	_ string,
) (*entities.Store, error) {
	return nil, nil
}

func (f *changeStoreImageRepository) ExistsBySlug(
	_ context.Context,
	_ string,
) (bool, error) {
	return false, nil
}

func (f *changeStoreImageRepository) ListActive(
	_ context.Context,
	_ string,
	_ int,
	_ int,
) ([]*entities.Store, int, error) {
	return nil, 0, nil
}

func (f *changeStoreImageRepository) Update(
	_ context.Context,
	store *entities.Store,
) error {
	f.updateCalled = true
	f.updatedStore = store
	return f.updateError
}

func (f *changeStoreImageRepository) ChangePlan(
	_ context.Context,
	_ *entities.Store,
) error {
	return nil
}

func (f *changeStoreImageRepository) ChangeStatus(
	_ context.Context,
	_ uuid.UUID,
	_ string,
) error {
	return nil
}

type changeStoreImageFormatResolver struct {
	called      bool
	contentType string
	extension   string
	err         error
}

func (f *changeStoreImageFormatResolver) ResolveExtension(
	_ context.Context,
	contentType string,
) (string, error) {
	f.called = true
	f.contentType = contentType
	return f.extension, f.err
}

type changeStoreImageDispatcher struct {
	called                       bool
	image                        ports.StoreImage
	dispatchError                error
	calledBeforeRepositoryUpdate bool
	repositoryUpdateObserved     bool
}

func (f *changeStoreImageDispatcher) DispatchStoreImage(
	_ context.Context,
	image ports.StoreImage,
) error {
	f.called = true
	f.image = image
	f.calledBeforeRepositoryUpdate = !f.repositoryUpdateObserved
	return f.dispatchError
}

type changeStoreImageLogger struct {
	successCalled bool
}

func (f *changeStoreImageLogger) Log(
	_ context.Context,
	event ports.LogEvent,
) error {
	if event.Event == "store.change_image.succeeded" {
		f.successCalled = true
	}

	return nil
}

type changeStoreImageMetrics struct {
	incrementCalls int
	observeCalls   int
}

func (f *changeStoreImageMetrics) Increment(
	_ context.Context,
	_ ports.Metric,
) error {
	f.incrementCalls++
	return nil
}

func (f *changeStoreImageMetrics) Observe(
	_ context.Context,
	_ ports.Metric,
) error {
	f.observeCalls++
	return nil
}
