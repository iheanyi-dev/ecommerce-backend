package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/value_objects"
)

// PaymentRepository persists payments and supports business-reference
// idempotency lookups.
type PaymentRepository interface {
	// GetByID retrieves a payment by its identifier.
	GetByID(ctx context.Context, id string) (*entities.Payment, error)

	// GetByBusinessReference retrieves a payment by its business reference.
	GetByBusinessReference(
		ctx context.Context,
		reference value_objects.BusinessReference,
	) (*entities.Payment, error)

	// Save inserts or updates a payment.
	// ListByPayerID returns a page of payments and the total matching count.
	ListByPayerID(ctx context.Context, payerID string, limit, offset int) ([]*entities.Payment, int64, error)
	Save(ctx context.Context, payment *entities.Payment) error
}
