package use_cases

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
)

// ChangeStoreImageUseCase replaces the image reference associated with a Store.
//
// The authenticated owner is obtained from the application authentication
// context. The caller does not provide the owner ID.
//
// The image is first dispatched to the Storage Service. The Store aggregate is
// only updated and persisted after the image dispatch succeeds. This prevents
// the persisted Store image reference from pointing at an image that the
// Storage Service rejected.
type ChangeStoreImageUseCase struct {
	storeRepository     ports.StoreRepository
	imageFormatResolver ports.ImageFormatResolver
	imageDispatcher     ports.ImageStorageDispatcher
	logger              ports.Logger
	metrics             ports.Metrics
}

// NewChangeStoreImageUseCase constructs the Change Store Image use case.
func NewChangeStoreImageUseCase(
	storeRepository ports.StoreRepository,
	imageFormatResolver ports.ImageFormatResolver,
	imageDispatcher ports.ImageStorageDispatcher,
	logger ports.Logger,
	metrics ports.Metrics,
) *ChangeStoreImageUseCase {
	return &ChangeStoreImageUseCase{
		storeRepository:     storeRepository,
		imageFormatResolver: imageFormatResolver,
		imageDispatcher:     imageDispatcher,
		logger:              logger,
		metrics:             metrics,
	}
}

// Execute replaces the Store image and persists the new image reference.
func (uc *ChangeStoreImageUseCase) Execute(
	ctx context.Context,
	input dto.ChangeStoreImageInput,
) (dto.ChangeStoreImageOutput, error) {
	startedAt := time.Now()

	uc.increment(ctx, ports.Metric{
		Name:  "store_change_image_total",
		Value: 1,
		Labels: map[string]string{
			"operation": "change_store_image",
		},
	})

	identity, ok := ports.AuthenticatedIdentityFromContext(ctx)
	if !ok || identity.UserID == uuid.Nil {
		return uc.fail(
			ctx,
			startedAt,
			identity,
			"unauthenticated",
			applicationerrors.ErrUnauthenticated,
		)
	}

	if input.Image == nil {
		return uc.fail(
			ctx,
			startedAt,
			identity,
			"invalid_image",
			fmt.Errorf("change store image: image is required"),
		)
	}

	store, err := uc.storeRepository.FindByID(ctx, input.StoreID)
	if err != nil {
		return uc.fail(
			ctx,
			startedAt,
			identity,
			"store_lookup",
			fmt.Errorf("find store: %w", err),
		)
	}

	if store == nil {
		return uc.fail(
			ctx,
			startedAt,
			identity,
			"store_not_found",
			applicationerrors.ErrStoreNotFound,
		)
	}

	// Ownership is enforced here rather than accepting an owner ID from the
	// request. Non-owners receive the same not-found error used by Update
	// Store so that Store existence is not unnecessarily disclosed.
	if !store.IsOwnedBy(identity.UserID) {
		return uc.fail(
			ctx,
			startedAt,
			identity,
			"store_not_found",
			applicationerrors.ErrStoreNotFound,
		)
	}

	extension, err := uc.imageFormatResolver.ResolveExtension(
		ctx,
		input.Image.ContentType,
	)
	if err != nil {
		return uc.fail(
			ctx,
			startedAt,
			identity,
			"image_format_resolution",
			fmt.Errorf("resolve image extension: %w", err),
		)
	}

	// Keep image naming identical to Create Store:
	//
	//	stores/{store_id}.{extension}
	//
	// The Store owns this reference and supplies it to the Storage Service.
	reference := fmt.Sprintf(
		"stores/%s.%s",
		store.ID().Value().String(),
		extension,
	)

	// Storage must succeed before the aggregate reference is changed.
	if err := uc.imageDispatcher.DispatchStoreImage(
		ctx,
		ports.StoreImage{
			StoreID:     store.ID().Value(),
			Reference:   reference,
			Content:     input.Image.Content,
			ContentType: input.Image.ContentType,
		},
	); err != nil {
		return uc.fail(
			ctx,
			startedAt,
			identity,
			"image_storage",
			fmt.Errorf("store image: %w", err),
		)
	}

	// Only mutate the aggregate after the Storage Service accepted the image.
	store.SetImageReference(reference)

	if err := uc.storeRepository.Update(ctx, store); err != nil {
		return uc.fail(
			ctx,
			startedAt,
			identity,
			"store_persistence",
			fmt.Errorf("update store: %w", err),
		)
	}

	output := toStoreOutput(store)

	_ = uc.logger.Log(ctx, ports.LogEvent{
		Event:     "store.change_image.succeeded",
		Operation: "change_store_image",
		UserID:    identity.UserID.String(),
		Role:      identity.Role,
	})

	uc.increment(ctx, ports.Metric{
		Name:  "store_change_image_success_total",
		Value: 1,
		Labels: map[string]string{
			"operation": "change_store_image",
		},
	})

	_ = uc.metrics.Observe(ctx, ports.Metric{
		Name:  "store_change_image_duration_seconds",
		Value: time.Since(startedAt).Seconds(),
		Labels: map[string]string{
			"operation": "change_store_image",
		},
	})

	return output, nil
}

// fail records failure observability while preserving the original
// application error returned to the caller.
func (uc *ChangeStoreImageUseCase) fail(
	ctx context.Context,
	startedAt time.Time,
	identity ports.AuthenticatedIdentity,
	failureCategory string,
	err error,
) (dto.ChangeStoreImageOutput, error) {
	_ = uc.logger.Log(ctx, ports.LogEvent{
		Event:           "store.change_image.failed",
		Operation:       "change_store_image",
		UserID:          identity.UserID.String(),
		Role:            identity.Role,
		FailureCategory: failureCategory,
	})

	uc.increment(ctx, ports.Metric{
		Name:  "store_change_image_failure_total",
		Value: 1,
		Labels: map[string]string{
			"operation":        "change_store_image",
			"failure_category": failureCategory,
		},
	})

	_ = uc.metrics.Observe(ctx, ports.Metric{
		Name:  "store_change_image_duration_seconds",
		Value: time.Since(startedAt).Seconds(),
		Labels: map[string]string{
			"operation": "change_store_image",
			"result":    "failure",
		},
	})

	return dto.ChangeStoreImageOutput{}, err
}

// increment keeps metrics failures from changing the outcome of the
// application operation.
func (uc *ChangeStoreImageUseCase) increment(
	ctx context.Context,
	metric ports.Metric,
) {
	_ = uc.metrics.Increment(ctx, metric)
}
