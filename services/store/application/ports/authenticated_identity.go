package ports

import (
	"context"

	"github.com/google/uuid"
)

// AuthenticatedIdentity represents the trusted identity established by the
// platform authentication boundary and propagated to the Store service.
type AuthenticatedIdentity struct {
	UserID uuid.UUID
	Role   string
}

type authenticatedIdentityContextKey struct{}

// WithAuthenticatedIdentity attaches a trusted authenticated identity to a
// context so application use cases can consume it without depending on HTTP
// or middleware-specific types.
func WithAuthenticatedIdentity(
	ctx context.Context,
	identity AuthenticatedIdentity,
) context.Context {
	return context.WithValue(ctx, authenticatedIdentityContextKey{}, identity)
}

// AuthenticatedIdentityFromContext retrieves the trusted authenticated
// identity from the context.
func AuthenticatedIdentityFromContext(
	ctx context.Context,
) (AuthenticatedIdentity, bool) {
	identity, ok := ctx.Value(authenticatedIdentityContextKey{}).(AuthenticatedIdentity)
	return identity, ok
}
