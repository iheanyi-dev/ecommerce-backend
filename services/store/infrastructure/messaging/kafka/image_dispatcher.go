package kafka

import (
	"context"
	"errors"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/messaging/events"
)

// ImageDispatcher publishes Store image storage requests to Kafka.
type ImageDispatcher struct {
	publisher MessagePublisher
}

// NewImageDispatcher creates a Store image dispatcher.
func NewImageDispatcher(publisher MessagePublisher) *ImageDispatcher {
	return &ImageDispatcher{
		publisher: publisher,
	}
}

// DispatchStoreImage publishes the complete Store image to the asynchronous
// image-storage topic.
//
// The Store application does not wait for Storage to persist the image.
// Successful publication means the asynchronous workflow has accepted the
// request for broker delivery.
func (d *ImageDispatcher) DispatchStoreImage(
	ctx context.Context,
	image ports.StoreImage,
) error {
	if image.StoreID == [16]byte{} {
		return errors.New("store image dispatcher: store ID is required")
	}

	if image.Reference == "" {
		return errors.New("store image dispatcher: image reference is required")
	}

	return d.publisher.Publish(
		ctx,
		events.StoreImageRequestedTopic,
		image.StoreID.String(),
		events.StoreImageRequested{
			StoreID:     image.StoreID,
			Reference:   image.Reference,
			ContentType: image.ContentType,
			Content:     image.Content,
		},
	)
}
