package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
)

// PayoutAttemptRepository persists provider attempts independently of payouts.
type PayoutAttemptRepository interface {
	GetByID(ctx context.Context, id string) (*entities.PayoutAttempt, error)
	GetByPayoutID(ctx context.Context, payoutID string) (*entities.PayoutAttempt, error)
	Save(ctx context.Context, attempt *entities.PayoutAttempt) error
}
