package dtos

// GetPromoForUserInput identifies the store, product, and customer
// for which the highest-priority active promotion is requested.
type GetPromoForUserInput struct {
	StoreID    string
	ProductID  string
	CustomerID string
}
