package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/activation"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// ActivationTokenRepository defines activation-token persistence operations.
type ActivationTokenRepository interface {
	// Create persists a new activation token.
	Create(ctx context.Context, token *activation.Token) error

	// FindByHash retrieves an activation token using its hash.
	FindByHash(ctx context.Context, tokenHash string) (*activation.Token, error)

	// Consume atomically marks an unconsumed token as consumed.
	// Implementations must reject tokens already consumed or expired.
	Consume(ctx context.Context, token *activation.Token) error

	// DeleteByUserID invalidates all outstanding activation tokens for a user.
	DeleteByUserID(ctx context.Context, userID user.UserID) error
}
