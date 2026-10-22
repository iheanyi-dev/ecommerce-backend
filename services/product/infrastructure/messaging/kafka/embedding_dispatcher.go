package kafka

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/infrastructure/messaging/events"
)

// EmbeddingDispatcher publishes Product embedding requests asynchronously.
type EmbeddingDispatcher struct {
	publisher MessagePublisher
}

func NewEmbeddingDispatcher(publisher MessagePublisher) *EmbeddingDispatcher {
	return &EmbeddingDispatcher{publisher: publisher}
}

func (d *EmbeddingDispatcher) DispatchProductEmbedding(ctx context.Context, product ports.ProductEmbedding) error {
	productID, err := uuid.Parse(product.ProductID)
	if err != nil || productID == uuid.Nil {
		return errors.New("product embedding dispatcher: valid product ID is required")
	}
	storeID, err := uuid.Parse(product.StoreID)
	if err != nil || storeID == uuid.Nil {
		return errors.New("product embedding dispatcher: valid store ID is required")
	}
	return d.publisher.Publish(ctx, events.ProductEmbeddingRequestedTopic, productID.String(),
		events.ProductEmbeddingRequested{
			ProductID: productID, StoreID: storeID,
			Name: product.Name, Description: product.Description,
		})
}

func (d *EmbeddingDispatcher) DispatchProductEmbeddingDeletion(ctx context.Context, productID, storeID string) error {
	id, err := uuid.Parse(productID)
	if err != nil || id == uuid.Nil {
		return errors.New("product embedding deletion dispatcher: valid product ID is required")
	}
	store, err := uuid.Parse(storeID)
	if err != nil || store == uuid.Nil {
		return errors.New("product embedding deletion dispatcher: valid store ID is required")
	}
	return d.publisher.Publish(ctx, events.ProductEmbeddingDeletionRequestedTopic, id.String(),
		events.ProductEmbeddingDeletionRequested{ProductID: id, StoreID: store})
}
