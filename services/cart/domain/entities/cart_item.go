package entities

import "github.com/google/uuid"

type CartItem struct {
	id        uuid.UUID
	productID uuid.UUID
	storeID   uuid.UUID
	quantity  int64
}

func newCartItem(productID, storeID uuid.UUID, quantity int64) *CartItem {
	return &CartItem{
		id:        uuid.New(),
		productID: productID,
		storeID:   storeID,
		quantity:  quantity,
	}
}

func (i *CartItem) ID() uuid.UUID        { return i.id }
func (i *CartItem) ProductID() uuid.UUID { return i.productID }
func (i *CartItem) StoreID() uuid.UUID   { return i.storeID }
func (i *CartItem) Quantity() int64      { return i.quantity }
