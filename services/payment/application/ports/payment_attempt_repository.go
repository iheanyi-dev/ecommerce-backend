package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
)

// PaymentAttemptRepository persists provider payment attempts.
type PaymentAttemptRepository interface {
	// GetByID retrieves an attempt by its identifier.
	GetByID(ctx context.Context, id string) (*entities.PaymentAttempt, error)

	// Save inserts or updates an attempt.
	Save(ctx context.Context, attempt *entities.PaymentAttempt) error
}
