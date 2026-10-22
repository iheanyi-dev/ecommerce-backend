package entities

import (
	"time"

	"github.com/google/uuid"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/errors"
)

type Cart struct {
	id         uuid.UUID
	customerID uuid.UUID
	items      []*CartItem
	createdAt  time.Time
	updatedAt  time.Time
}

func NewCart(customerID uuid.UUID) (*Cart, error) {
	if customerID == uuid.Nil {
		return nil, domainerrors.ErrInvalidCustomerID
	}

	now := time.Now().UTC()
	return &Cart{
		id:         uuid.New(),
		customerID: customerID,
		items:      make([]*CartItem, 0),
		createdAt:  now,
		updatedAt:  now,
	}, nil
}

func (c *Cart) ID() uuid.UUID         { return c.id }
func (c *Cart) CustomerID() uuid.UUID { return c.customerID }
func (c *Cart) CreatedAt() time.Time  { return c.createdAt }
func (c *Cart) UpdatedAt() time.Time  { return c.updatedAt }

func (c *Cart) Items() []*CartItem {
	items := make([]*CartItem, len(c.items))
	copy(items, c.items)
	return items
}

func (c *Cart) AddItem(productID, storeID uuid.UUID, quantity int64) error {
	if productID == uuid.Nil {
		return domainerrors.ErrInvalidProductID
	}
	if storeID == uuid.Nil {
		return domainerrors.ErrInvalidStoreID
	}
	if quantity <= 0 {
		return domainerrors.ErrInvalidQuantity
	}

	for _, item := range c.items {
		if item.productID != productID {
			continue
		}
		if item.storeID != storeID {
			return domainerrors.ErrProductStoreMismatch
		}
		item.quantity = quantity
		c.touch()
		return nil
	}

	c.items = append(c.items, newCartItem(productID, storeID, quantity))
	c.touch()
	return nil
}

func (c *Cart) SetItemQuantity(itemID uuid.UUID, quantity int64) error {
	if quantity <= 0 {
		return domainerrors.ErrInvalidQuantity
	}

	for _, item := range c.items {
		if item.id == itemID {
			item.quantity = quantity
			c.touch()
			return nil
		}
	}

	return domainerrors.ErrCartItemNotFound
}

// ReconcileItem applies authoritative Product service data during checkout.
// Unlike a customer-requested quantity update, checkout reconciliation allows
// zero when the product has no stock. Negative quantities remain invalid.
func (c *Cart) ReconcileItem(itemID, storeID uuid.UUID, quantity int64) error {
	if storeID == uuid.Nil {
		return domainerrors.ErrInvalidStoreID
	}
	if quantity < 0 {
		return domainerrors.ErrInvalidQuantity
	}

	for _, item := range c.items {
		if item.id != itemID {
			continue
		}

		if item.storeID != storeID || item.quantity != quantity {
			item.storeID = storeID
			item.quantity = quantity
			c.touch()
		}
		return nil
	}

	return domainerrors.ErrCartItemNotFound
}

func (c *Cart) RemoveItem(itemID uuid.UUID) error {
	for index, item := range c.items {
		if item.id == itemID {
			c.items = append(c.items[:index], c.items[index+1:]...)
			c.touch()
			return nil
		}
	}

	return domainerrors.ErrCartItemNotFound
}

func (c *Cart) Clear() {
	if len(c.items) == 0 {
		return
	}

	c.items = make([]*CartItem, 0)
	c.touch()
}

func (c *Cart) Validate() error {
	if c.id == uuid.Nil {
		return domainerrors.ErrInvalidCartID
	}
	if c.customerID == uuid.Nil {
		return domainerrors.ErrInvalidCustomerID
	}

	for _, item := range c.items {
		if item == nil || item.id == uuid.Nil {
			return domainerrors.ErrCartItemNotFound
		}
		if item.productID == uuid.Nil {
			return domainerrors.ErrInvalidProductID
		}
		if item.storeID == uuid.Nil {
			return domainerrors.ErrInvalidStoreID
		}
		if item.quantity < 0 {
			return domainerrors.ErrInvalidQuantity
		}
	}

	return nil
}

func (c *Cart) touch() {
	c.updatedAt = time.Now().UTC()
}

// CartItemState contains the persisted state needed to reconstitute a cart item.
// It is a domain-level input, not a storage-specific representation.
type CartItemState struct {
	ID        uuid.UUID
	ProductID uuid.UUID
	StoreID   uuid.UUID
	Quantity  int64
}

// ReconstituteCart restores an existing cart without generating new IDs or
// timestamps. Unlike NewCart and AddItem, it preserves persisted item order
// and permits zero-quantity items produced by checkout reconciliation.
func ReconstituteCart(
	id uuid.UUID,
	customerID uuid.UUID,
	items []CartItemState,
	createdAt time.Time,
	updatedAt time.Time,
) (*Cart, error) {
	cart := &Cart{
		id:         id,
		customerID: customerID,
		items:      make([]*CartItem, 0, len(items)),
		createdAt:  createdAt,
		updatedAt:  updatedAt,
	}

	for _, item := range items {
		cart.items = append(cart.items, &CartItem{
			id:        item.ID,
			productID: item.ProductID,
			storeID:   item.StoreID,
			quantity:  item.Quantity,
		})
	}

	if err := cart.Validate(); err != nil {
		return nil, err
	}

	return cart, nil
}
