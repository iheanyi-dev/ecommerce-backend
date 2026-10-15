package ports

import "context"

// ProductEmbedding contains the product data needed to request an embedding.
type ProductEmbedding struct {
	ProductID   string
	StoreID     string
	Name        string
	Description string
}

// EmbeddingDispatcher asynchronously requests embedding creation or deletion.
// Dispatch failures must not cause an already-persisted product to be rolled back.
type EmbeddingDispatcher interface {
	DispatchProductEmbedding(ctx context.Context, product ProductEmbedding) error
	DispatchProductEmbeddingDeletion(ctx context.Context, productID, storeID string) error
}
