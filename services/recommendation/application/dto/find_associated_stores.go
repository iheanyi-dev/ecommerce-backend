package dto

import "github.com/google/uuid"

// FindAssociatedStoresInput contains the Store and page for behavioral association search.
type FindAssociatedStoresInput struct {
	StoreID uuid.UUID
	Page    int
}

// FindAssociatedStoresOutput contains the requested page of associated Stores.
type FindAssociatedStoresOutput struct {
	Recommendations []StoreRecommendation
}
