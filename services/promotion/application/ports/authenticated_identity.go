package ports

import (
	"context"

	"github.com/google/uuid"
)

type AuthenticatedIdentity struct {
	UserID uuid.UUID
	Role   string
}

type authenticatedIdentityContextKey struct{}

func WithAuthenticatedIdentity(
	ctx context.Context,
	identity AuthenticatedIdentity,
) context.Context {
	return context.WithValue(ctx, authenticatedIdentityContextKey{}, identity)
}

func AuthenticatedIdentityFromContext(
	ctx context.Context,
) (AuthenticatedIdentity, bool) {
	identity, ok := ctx.Value(authenticatedIdentityContextKey{}).(AuthenticatedIdentity)
	return identity, ok
}
