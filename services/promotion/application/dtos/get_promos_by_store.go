package dtos

// GetPromosByStoreInput contains the store and pagination criteria.
type GetPromosByStoreInput struct {
	StoreID string
	Limit   int
	Offset  int
}
