package ports

import (
	"context"

	"github.com/google/uuid"
)

// IdentityClient retrieves the authenticated customer's identity information.
type IdentityClient interface {
	GetUserEmail(ctx context.Context, userID uuid.UUID) (string, error)
}
