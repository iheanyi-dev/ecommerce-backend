package ports

import (
	"context"

	"github.com/google/uuid"
)

// IdentityUser contains the identity information Store needs from Identity.
type IdentityUser struct {
	ID    uuid.UUID
	Email string
}

// IdentityProvider exposes the Identity capabilities required by the Store
// application layer.
//
// The application depends on these business capabilities rather than on a
// concrete Identity Service client or communication technology. Infrastructure
// provides the implementation.
type IdentityProvider interface {
	// GetUser retrieves a user's identity information from the Identity Service.
	GetUser(ctx context.Context, userID uuid.UUID) (*IdentityUser, error)

	// IsVendor determines whether the supplied user currently has vendor
	// status in the Identity Service.
	IsVendor(ctx context.Context, userID uuid.UUID) (bool, error)

	// PromoteToVendor changes the supplied user's role to vendor in the
	// Identity Service.
	//
	// This operation is performed synchronously during Store creation because
	// a successfully created Store must have a corresponding vendor identity.
	PromoteToVendor(ctx context.Context, userID uuid.UUID) error
}
