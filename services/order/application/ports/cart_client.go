package ports

import (
	"context"

	"github.com/google/uuid"
)

// CartCheckoutItem contains the authoritative product data returned by Cart.
type CartCheckoutItem struct {
	ProductID  uuid.UUID
	Name       string
	Quantity   int64
	PriceMinor int64
	Currency   string
}

// CartCheckoutStore groups checkout items belonging to one store.
type CartCheckoutStore struct {
	StoreID uuid.UUID
	Items   []CartCheckoutItem
}

// CartCheckoutOutput is the reconciled cart snapshot used to create orders.
type CartCheckoutOutput struct {
	CartID            uuid.UUID
	Stores            []CartCheckoutStore
	RemovedProductIDs []uuid.UUID
}

// CartClient retrieves and reconciles a customer's specified cart.
type CartClient interface {
	Checkout(ctx context.Context, customerID, cartID uuid.UUID) (*CartCheckoutOutput, error)
}
