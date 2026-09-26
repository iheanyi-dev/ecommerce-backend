package kafka

import (
	"context"
	"errors"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/messaging/events"
)

// ImageDeletionDispatcher publishes obsolete Store image deletion requests.
type ImageDeletionDispatcher struct {
	publisher MessagePublisher
}

// NewImageDeletionDispatcher creates a Store image deletion dispatcher.
func NewImageDeletionDispatcher(
	publisher MessagePublisher,
) *ImageDeletionDispatcher {
	return &ImageDeletionDispatcher{
		publisher: publisher,
	}
}

// DispatchStoreImageDeletion publishes an asynchronous deletion request.
//
// Deletion is deliberately asynchronous because failure to clean up the old
// object must not make an otherwise successful image replacement fail.
func (d *ImageDeletionDispatcher) DispatchStoreImageDeletion(
	ctx context.Context,
	image ports.StoreImageDeletion,
) error {
	if image.StoreID == [16]byte{} {
		return errors.New("store image deletion dispatcher: store ID is required")
	}

	if image.Reference == "" {
		return errors.New("store image deletion dispatcher: image reference is required")
	}

	return d.publisher.Publish(
		ctx,
		events.StoreImageDeletionRequestedTopic,
		image.StoreID.String(),
		events.StoreImageDeletionRequested{
			StoreID:   image.StoreID,
			Reference: image.Reference,
		},
	)
}
