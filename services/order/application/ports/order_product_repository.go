package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/order/domain/entities"
)

// OrderProductRepository batch-persists order lines within the supplied transaction.
type OrderProductRepository interface {
	// CreateMany batch-persists order lines within the supplied transaction.
	CreateMany(ctx context.Context, products []*entities.OrderProduct) error

	// FindByOrderIDs returns product lines belonging to any of the supplied orders.
	FindByOrderIDs(ctx context.Context, orderIDs []uuid.UUID) ([]*entities.OrderProduct, error)
}
