package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/entities"
)

// CartRepository defines the persistence operations required by the Cart
// application. Implementations may use Redis or another persistence system.
type CartRepository interface {
	// GetByCustomerID retrieves the customer's cart.
	// It returns ErrCartNotFound when the customer has no cart.
	GetByCustomerID(ctx context.Context, customerID uuid.UUID) (*entities.Cart, error)

	// Save creates or replaces the customer's cart.
	Save(ctx context.Context, cart *entities.Cart) error

	// DeleteByCustomerID removes the customer's cart if it exists.
	DeleteByCustomerID(ctx context.Context, customerID uuid.UUID) error
}
