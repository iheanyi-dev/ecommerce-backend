package ports

import (
	"context"

	"github.com/google/uuid"
)

// AuthenticatedIdentity contains identity propagated by the trusted Gateway.
type AuthenticatedIdentity struct {
	UserID uuid.UUID
	Role   string
}

type authenticatedIdentityContextKey struct{}

// WithAuthenticatedIdentity attaches a trusted identity to a context.
func WithAuthenticatedIdentity(
	ctx context.Context,
	identity AuthenticatedIdentity,
) context.Context {
	return context.WithValue(ctx, authenticatedIdentityContextKey{}, identity)
}

// AuthenticatedIdentityFromContext retrieves the propagated identity.
func AuthenticatedIdentityFromContext(
	ctx context.Context,
) (AuthenticatedIdentity, bool) {
	identity, ok := ctx.Value(authenticatedIdentityContextKey{}).(AuthenticatedIdentity)
	return identity, ok
}
