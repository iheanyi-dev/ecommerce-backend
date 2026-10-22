package ports

import (
	"context"

	"github.com/google/uuid"
)

// AuthenticatedIdentity contains the trusted identity propagated by the Gateway.
type AuthenticatedIdentity struct {
	UserID uuid.UUID
	Role   string
}

type authenticatedIdentityContextKey struct{}

// WithAuthenticatedIdentity attaches a trusted identity to the context.
func WithAuthenticatedIdentity(ctx context.Context, identity AuthenticatedIdentity) context.Context {
	return context.WithValue(ctx, authenticatedIdentityContextKey{}, identity)
}

// AuthenticatedIdentityFromContext retrieves the trusted identity, if present.
func AuthenticatedIdentityFromContext(ctx context.Context) (AuthenticatedIdentity, bool) {
	identity, ok := ctx.Value(authenticatedIdentityContextKey{}).(AuthenticatedIdentity)
	return identity, ok
}
