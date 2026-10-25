package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/domain/entities"
)

// OrderRepository defines persistence operations for creating and managing orders.
type OrderRepository interface {
	// CreateMany persists multiple orders as part of the caller's transaction.
	CreateMany(ctx context.Context, orders []*entities.Order) error

	// FindByID retrieves an existing order by its unique identifier.
	FindByID(ctx context.Context, orderID uuid.UUID) (*entities.Order, error)

	// Update persists changes to an existing order after a valid domain transition.
	Update(ctx context.Context, order *entities.Order) error

	// ListByCustomerID returns one page of a customer's orders and the total match count.
	ListByCustomerID(ctx context.Context, customerID uuid.UUID, offset, limit int) ([]*entities.Order, int64, error)

	// ListByStoreID returns one page of a store's orders and the total match count.
	ListByStoreID(ctx context.Context, storeID uuid.UUID, offset, limit int) ([]*entities.Order, int64, error)
}
