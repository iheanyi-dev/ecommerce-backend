package ports

import (
	"context"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/activation"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// ActivationTokenRepository defines activation-token persistence operations.
type ActivationTokenRepository interface {
	// Create persists a new activation token.
	Create(ctx context.Context, token *activation.Token) error

	// DeleteByUserID invalidates all activation tokens belonging to a user.
	DeleteByUserID(ctx context.Context, userID user.UserID) error

	// ConsumeAndActivate atomically consumes a valid token and activates
	// its pending user account. Both changes must commit or roll back together.
	ConsumeAndActivate(ctx context.Context, tokenHash string, at time.Time) error
}
