package entities

import (
	"strings"

	"github.com/google/uuid"
	domain "github.com/iheanyi-dev/ecommerce-backend/services/order/domain"
)

// OrderProduct is a purchased product snapshot stored independently.
// ID identifies this order line; ProductID identifies the catalog product.
type OrderProduct struct {
	ID        uuid.UUID
	OrderID   uuid.UUID
	ProductID uuid.UUID
	Name      string
	Qty       int64
	Price     int64
}

// NewOrderProduct creates a validated product snapshot linked to an order.
func NewOrderProduct(
	id uuid.UUID,
	orderID uuid.UUID,
	productID uuid.UUID,
	name string,
	qty int64,
	price int64,
) (OrderProduct, error) {
	if id == uuid.Nil || orderID == uuid.Nil || productID == uuid.Nil {
		return OrderProduct{}, domain.ErrInvalidID
	}
	if strings.TrimSpace(name) == "" {
		return OrderProduct{}, domain.ErrInvalidProduct
	}
	if qty <= 0 {
		return OrderProduct{}, domain.ErrInvalidQuantity
	}
	if price < 0 {
		return OrderProduct{}, domain.ErrInvalidPrice
	}

	return OrderProduct{
		ID:        id,
		OrderID:   orderID,
		ProductID: productID,
		Name:      strings.TrimSpace(name),
		Qty:       qty,
		Price:     price,
	}, nil
}
