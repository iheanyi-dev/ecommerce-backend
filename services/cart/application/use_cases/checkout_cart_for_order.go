package use_cases

import (
	"context"

	"github.com/google/uuid"
)

// CheckoutCartForOrder is the internal application entry point used by Order.
// It validates customer/cart identity before reconciling the cart and returning
// the authoritative product snapshot.
type CheckoutCartForOrder struct {
	checkout *CheckoutCart
}

func NewCheckoutCartForOrder(checkout *CheckoutCart) *CheckoutCartForOrder {
	return &CheckoutCartForOrder{checkout: checkout}
}

func (uc *CheckoutCartForOrder) Execute(
	ctx context.Context,
	customerID, cartID uuid.UUID,
) (*CheckoutCartOutput, error) {
	return uc.checkout.ExecuteForCart(ctx, customerID, cartID)
}
