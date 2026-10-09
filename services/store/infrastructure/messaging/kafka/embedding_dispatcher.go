package kafka

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/messaging/events"
)

// EmbeddingDispatcher publishes Store embedding work to Kafka.
type EmbeddingDispatcher struct {
	publisher MessagePublisher
}

// NewEmbeddingDispatcher creates a Store embedding dispatcher.
func NewEmbeddingDispatcher(publisher MessagePublisher) *EmbeddingDispatcher {
	return &EmbeddingDispatcher{
		publisher: publisher,
	}
}

// DispatchStoreEmbedding publishes the current Store profile to Kafka.
func (d *EmbeddingDispatcher) DispatchStoreEmbedding(
	ctx context.Context,
	embedding ports.StoreEmbedding,
) error {
	if embedding.StoreID == uuid.Nil {
		return errors.New("store embedding dispatcher: store ID is required")
	}

	return d.publisher.Publish(
		ctx,
		events.StoreEmbeddingRequestedTopic,
		embedding.StoreID.String(),
		events.StoreEmbeddingRequested{
			StoreID:     embedding.StoreID,
			Name:        embedding.Name,
			Description: embedding.Description,
		},
	)
}

// DispatchStoreEmbeddingDeletion publishes a Store embedding deletion request.
func (d *EmbeddingDispatcher) DispatchStoreEmbeddingDeletion(
	ctx context.Context,
	storeID uuid.UUID,
) error {
	if storeID == uuid.Nil {
		return errors.New("store embedding deletion dispatcher: store ID is required")
	}

	return d.publisher.Publish(
		ctx,
		events.StoreEmbeddingDeletionRequestedTopic,
		storeID.String(),
		events.StoreEmbeddingDeletionRequested{
			StoreID: storeID,
		},
	)
}
