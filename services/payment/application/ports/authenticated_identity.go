package ports

import (
	"context"

	"github.com/google/uuid"
)

// AuthenticatedIdentity represents the trusted identity established by the
// platform authentication boundary and propagated to the Payment service.
type AuthenticatedIdentity struct {
	UserID uuid.UUID
	Role   string
}

// authenticatedIdentityContextKey prevents collisions with context keys
// defined by other packages.
type authenticatedIdentityContextKey struct{}

// WithAuthenticatedIdentity attaches a trusted identity to the context so
// application use cases can consume it without depending on HTTP middleware.
func WithAuthenticatedIdentity(
	ctx context.Context,
	identity AuthenticatedIdentity,
) context.Context {
	return context.WithValue(ctx, authenticatedIdentityContextKey{}, identity)
}

// AuthenticatedIdentityFromContext retrieves the trusted identity from
// the context. The boolean reports whether an identity was present.
func AuthenticatedIdentityFromContext(
	ctx context.Context,
) (AuthenticatedIdentity, bool) {
	identity, ok := ctx.Value(authenticatedIdentityContextKey{}).(AuthenticatedIdentity)
	return identity, ok
}
