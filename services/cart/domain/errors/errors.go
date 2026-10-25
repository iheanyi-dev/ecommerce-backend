package errors

import "errors"

var (
	ErrCartNotFound          = errors.New("cart not found")
	ErrCartOwnershipMismatch = errors.New("cart does not belong to customer")
	ErrInvalidCustomerID     = errors.New("invalid customer ID")
	ErrInvalidCartID         = errors.New("invalid cart ID")
	ErrInvalidProductID      = errors.New("invalid product ID")
	ErrInvalidStoreID        = errors.New("invalid store ID")
	ErrInvalidQuantity       = errors.New("invalid quantity")
	ErrCartItemNotFound      = errors.New("cart item not found")
	ErrProductStoreMismatch  = errors.New("product is associated with a different store")
)

// ErrCartNotFound indicates that the customer does not have a persisted cart.
