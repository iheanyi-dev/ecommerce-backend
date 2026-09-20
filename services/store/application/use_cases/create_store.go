package use_cases

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
)

// CreateStoreUseCase coordinates Store creation and the synchronous
// Store/Identity consistency workflow.
//
// The use case deliberately does not know about HTTP, gRPC, Kafka, Redpanda,
// storage implementations, or recommendation implementations. Those concerns
// are represented by application ports and implemented in infrastructure.
//
// Tracing remains outside the use case. The context is propagated so
// presentation/middleware-level request correlation can flow through the
// application and infrastructure layers.
type CreateStoreUseCase struct {
	storeRepository     ports.StoreRepository
	identityProvider    ports.IdentityProvider
	imageFormatResolver ports.ImageFormatResolver
	imageDispatcher     ports.ImageStorageDispatcher
	embeddingDispatcher ports.EmbeddingDispatcher
	logger              ports.Logger
	metrics             ports.Metrics
}

// NewCreateStoreUseCase constructs the Create Store application workflow.
func NewCreateStoreUseCase(
	storeRepository ports.StoreRepository,
	identityProvider ports.IdentityProvider,
	imageFormatResolver ports.ImageFormatResolver,
	imageDispatcher ports.ImageStorageDispatcher,
	embeddingDispatcher ports.EmbeddingDispatcher,
	logger ports.Logger,
	metrics ports.Metrics,
) *CreateStoreUseCase {
	return &CreateStoreUseCase{
		storeRepository:     storeRepository,
		identityProvider:    identityProvider,
		imageFormatResolver: imageFormatResolver,
		imageDispatcher:     imageDispatcher,
		embeddingDispatcher: embeddingDispatcher,
		logger:              logger,
		metrics:             metrics,
	}
}

// Execute creates a Store for the authenticated user.
//
// The synchronous success invariant is:
//
//	Store exists + Identity identifies the owner as a vendor.
//
// Image storage and recommendation embedding work are dispatched only after
// that invariant has been established. Failure to dispatch either asynchronous
// operation does not invalidate Store creation.
func (uc *CreateStoreUseCase) Execute(
	ctx context.Context,
	input dto.CreateStoreInput,
) (dto.CreateStoreOutput, error) {
	startedAt := time.Now()

	uc.incrementMetric(ctx, ports.Metric{
		Name:  "store_create_total",
		Value: 1,
		Labels: map[string]string{
			"operation": "create_store",
		},
	})

	identity, ok := ports.AuthenticatedIdentityFromContext(ctx)
	if !ok || identity.UserID == uuid.Nil {
		return uc.fail(
			ctx,
			startedAt,
			"unauthenticated",
			applicationerrors.ErrUnauthenticated,
		)
	}

	isVendor, err := uc.identityProvider.IsVendor(ctx, identity.UserID)
	if err != nil {
		return uc.fail(
			ctx,
			startedAt,
			"identity_check",
			fmt.Errorf("check vendor status: %w", err),
		)
	}

	store, err := uc.buildStore(ctx, identity.UserID, input)
	if err != nil {
		return uc.fail(
			ctx,
			startedAt,
			"store_creation",
			err,
		)
	}

	err = uc.storeRepository.Create(ctx, store)
	if err != nil {
		if errors.Is(err, applicationerrors.ErrStoreAlreadyExists) && !isVendor {
			return uc.reconcileExistingStore(
				ctx,
				startedAt,
				identity.UserID,
				input,
			)
		}

		if errors.Is(err, applicationerrors.ErrStoreAlreadyExists) {
			return uc.fail(
				ctx,
				startedAt,
				"store_already_exists",
				applicationerrors.ErrStoreAlreadyExists,
			)
		}

		return uc.fail(
			ctx,
			startedAt,
			"store_persistence",
			fmt.Errorf("persist store: %w", err),
		)
	}

	// Identity promotion is the second half of the synchronous consistency
	// invariant. It is only required when Identity did not already identify the
	// owner as a vendor.
	if !isVendor {
		if err := uc.identityProvider.PromoteToVendor(ctx, identity.UserID); err != nil {
			if deleteErr := uc.storeRepository.Delete(ctx, store.ID().Value()); deleteErr != nil {
				return uc.fail(
					ctx,
					startedAt,
					"store_compensation",
					fmt.Errorf(
						"identity promotion failed: %w; store compensation failed: %v",
						err,
						deleteErr,
					),
				)
			}

			return uc.fail(
				ctx,
				startedAt,
				"identity_promotion",
				fmt.Errorf("promote owner to vendor: %w", err),
			)
		}
	}

	// At this point the synchronous success invariant has been established.
	// Async work is deliberately best-effort from this workflow's perspective.
	uc.dispatchImage(ctx, store, input.Image)
	uc.dispatchEmbedding(ctx, store)

	output := toStoreOutput(store)

	uc.log(ctx, ports.LogEvent{
		Event:     "store.create.succeeded",
		Operation: "create_store",
		UserID:    identity.UserID.String(),
		Role:      identity.Role,
	})

	uc.incrementMetric(ctx, ports.Metric{
		Name:  "store_create_success_total",
		Value: 1,
		Labels: map[string]string{
			"operation": "create_store",
		},
	})

	uc.observeMetric(ctx, startedAt)

	return output, nil
}

// buildStore constructs the domain aggregate and resolves the image reference
// before persistence. ContentType, rather than the supplied filename, is the
// authoritative source for the stored extension.
func (uc *CreateStoreUseCase) buildStore(
	ctx context.Context,
	ownerID uuid.UUID,
	input dto.CreateStoreInput,
) (*entities.Store, error) {
	store, err := entities.NewStore(
		ownerID,
		input.Name,
		input.Slug,
		input.Description,
		nil,
		string(valueobjects.PlanTypeBasic),
	)
	if err != nil {
		return nil, err
	}

	if input.Image == nil {
		return store, nil
	}

	extension, err := uc.imageFormatResolver.ResolveExtension(
		ctx,
		input.Image.ContentType,
	)
	if err != nil {
		return nil, fmt.Errorf("resolve image format: %w", err)
	}

	if extension == "" {
		return nil, errors.New("resolve image format: empty extension")
	}

	reference := fmt.Sprintf(
		"stores/%s.%s",
		store.ID().Value().String(),
		extension,
	)

	store.SetImageReference(reference)

	return store, nil
}

// reconcileExistingStore handles the specific inconsistent state where
// Identity says the user is not yet a vendor but the Store repository reports
// that a Store already exists for that owner.
//
// This can happen when a previous workflow persisted the Store successfully
// but failed before Identity promotion. The next relevant Create Store
// workflow reconciles that state instead of creating another Store.
func (uc *CreateStoreUseCase) reconcileExistingStore(
	ctx context.Context,
	startedAt time.Time,
	ownerID uuid.UUID,
	input dto.CreateStoreInput,
) (dto.CreateStoreOutput, error) {
	existingStore, err := uc.storeRepository.FindByOwnerID(ctx, ownerID)
	if err != nil {
		return uc.fail(
			ctx,
			startedAt,
			"store_reconciliation_lookup",
			fmt.Errorf("find existing store for reconciliation: %w", err),
		)
	}

	if existingStore == nil {
		return uc.fail(
			ctx,
			startedAt,
			"store_reconciliation_lookup",
			applicationerrors.ErrStoreNotFound,
		)
	}

	if err := uc.identityProvider.PromoteToVendor(ctx, ownerID); err != nil {
		if deleteErr := uc.storeRepository.Delete(
			ctx,
			existingStore.ID().Value(),
		); deleteErr != nil {
			return uc.fail(
				ctx,
				startedAt,
				"store_compensation",
				fmt.Errorf(
					"identity promotion failed: %w; store compensation failed: %v",
					err,
					deleteErr,
				),
			)
		}

		return uc.fail(
			ctx,
			startedAt,
			"identity_promotion",
			fmt.Errorf("promote existing store owner to vendor: %w", err),
		)
	}

	// The existing Store is the authoritative persisted aggregate. Do not use
	// the newly constructed aggregate from the failed creation attempt.
	uc.dispatchImage(ctx, existingStore, input.Image)
	uc.dispatchEmbedding(ctx, existingStore)

	output := toStoreOutput(existingStore)

	uc.log(ctx, ports.LogEvent{
		Event:     "store.create.succeeded",
		Operation: "create_store",
		UserID:    ownerID.String(),
	})

	uc.incrementMetric(ctx, ports.Metric{
		Name:  "store_create_success_total",
		Value: 1,
		Labels: map[string]string{
			"operation":  "create_store",
			"reconciled": "true",
		},
	})

	uc.observeMetric(ctx, startedAt)

	return output, nil
}

// dispatchImage submits image storage work only when both the request contains
// an image and the persisted Store has an image reference.
//
// A dispatcher error is observable but intentionally does not fail Store
// creation because image storage is asynchronous.
func (uc *CreateStoreUseCase) dispatchImage(
	ctx context.Context,
	store *entities.Store,
	image *dto.StoreImageInput,
) {
	if image == nil || store.ImageReference() == nil {
		return
	}

	err := uc.imageDispatcher.DispatchStoreImage(ctx, ports.StoreImage{
		StoreID:     store.ID().Value(),
		Reference:   *store.ImageReference(),
		Content:     image.Content,
		ContentType: image.ContentType,
	})
	if err == nil {
		return
	}

	uc.log(ctx, ports.LogEvent{
		Event:           "store.create.image_dispatch_failed",
		Operation:       "create_store",
		UserID:          store.OwnerID().Value().String(),
		FailureCategory: "image_dispatch",
	})

	uc.incrementMetric(ctx, ports.Metric{
		Name:  "store_create_async_dispatch_failure_total",
		Value: 1,
		Labels: map[string]string{
			"operation": "create_store",
			"dispatch":  "image",
		},
	})
}

// dispatchEmbedding submits recommendation embedding work after the
// synchronous Store/Identity invariant has succeeded.
func (uc *CreateStoreUseCase) dispatchEmbedding(
	ctx context.Context,
	store *entities.Store,
) {
	err := uc.embeddingDispatcher.DispatchStoreEmbedding(ctx, ports.StoreEmbedding{
		StoreID:     store.ID().Value(),
		Name:        store.Name().Value(),
		Description: store.Description().Value(),
	})
	if err == nil {
		return
	}

	uc.log(ctx, ports.LogEvent{
		Event:           "store.create.embedding_dispatch_failed",
		Operation:       "create_store",
		UserID:          store.OwnerID().Value().String(),
		FailureCategory: "embedding_dispatch",
	})

	uc.incrementMetric(ctx, ports.Metric{
		Name:  "store_create_async_dispatch_failure_total",
		Value: 1,
		Labels: map[string]string{
			"operation": "create_store",
			"dispatch":  "embedding",
		},
	})
}

// fail records a failed Create Store operation. Observability failures are
// deliberately ignored so logging/metrics can never change business behavior.
func (uc *CreateStoreUseCase) fail(
	ctx context.Context,
	startedAt time.Time,
	failureCategory string,
	err error,
) (dto.CreateStoreOutput, error) {
	identity, _ := ports.AuthenticatedIdentityFromContext(ctx)

	uc.log(ctx, ports.LogEvent{
		Event:           "store.create.failed",
		Operation:       "create_store",
		UserID:          identity.UserID.String(),
		Role:            identity.Role,
		FailureCategory: failureCategory,
	})

	uc.incrementMetric(ctx, ports.Metric{
		Name:  "store_create_failure_total",
		Value: 1,
		Labels: map[string]string{
			"operation":        "create_store",
			"failure_category": failureCategory,
		},
	})

	uc.observeMetric(ctx, startedAt)

	return dto.CreateStoreOutput{}, err
}

func (uc *CreateStoreUseCase) log(
	ctx context.Context,
	event ports.LogEvent,
) {
	if uc.logger == nil {
		return
	}

	_ = uc.logger.Log(ctx, event)
}

func (uc *CreateStoreUseCase) incrementMetric(
	ctx context.Context,
	metric ports.Metric,
) {
	if uc.metrics == nil {
		return
	}

	_ = uc.metrics.Increment(ctx, metric)
}

func (uc *CreateStoreUseCase) observeMetric(
	ctx context.Context,
	startedAt time.Time,
) {
	if uc.metrics == nil {
		return
	}

	_ = uc.metrics.Observe(ctx, ports.Metric{
		Name:  "store_create_duration_seconds",
		Value: time.Since(startedAt).Seconds(),
		Labels: map[string]string{
			"operation": "create_store",
		},
	})
}

func toStoreOutput(store *entities.Store) dto.CreateStoreOutput {
	return dto.StoreOutput{
		ID:             store.ID(),
		OwnerID:        store.OwnerID(),
		Name:           store.Name(),
		Slug:           store.Slug(),
		Description:    store.Description(),
		ImageReference: store.ImageReference(),
		Status:         store.Status(),
		Plan:           store.Plan(),
		CreatedAt:      store.CreatedAt(),
		UpdatedAt:      store.UpdatedAt(),
	}
}
