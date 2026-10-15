// Package ports defines outbound contracts required by Product application use cases.
package ports

import "context"

// StoreClient exposes Store operations required by the Product service.
// IncrementProductCount must atomically reserve capacity and reject the
// reservation when the store has reached its product limit.
type StoreClient interface {
	// GetStoreIDByOwnerID returns the owner's store ID. An empty ID with no
	// error means the owner has no store; dependency failures return errors.
	GetStoreIDByOwnerID(ctx context.Context, ownerID string) (string, error)

	IncrementProductCount(ctx context.Context, storeID string) error
	DecrementProductCount(ctx context.Context, storeID string) error

	// IsStoreActive reports whether a store is active and available to customers.
	// Dependency failures must be returned as errors, not treated as inactivity.
	IsStoreActive(ctx context.Context, storeID string) (bool, error)
	// FilterActiveStoreIDs returns only active IDs from the supplied store IDs.
	FilterActiveStoreIDs(ctx context.Context, storeIDs []string) ([]string, error)
}
