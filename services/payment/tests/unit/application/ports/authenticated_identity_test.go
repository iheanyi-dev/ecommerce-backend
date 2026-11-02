package ports_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
)

func TestAuthenticatedIdentityContext(t *testing.T) {
	userID := uuid.New()
	identity := ports.AuthenticatedIdentity{
		UserID: userID,
		Role:   "customer",
	}

	ctx := ports.WithAuthenticatedIdentity(context.Background(), identity)

	got, ok := ports.AuthenticatedIdentityFromContext(ctx)
	if !ok {
		t.Fatal("expected authenticated identity in context")
	}

	if got.UserID != userID {
		t.Fatalf("expected user ID %s, got %s", userID, got.UserID)
	}

	if got.Role != identity.Role {
		t.Fatalf("expected role %q, got %q", identity.Role, got.Role)
	}
}

func TestAuthenticatedIdentityFromContextWhenMissing(t *testing.T) {
	_, ok := ports.AuthenticatedIdentityFromContext(context.Background())
	if ok {
		t.Fatal("expected no authenticated identity in empty context")
	}
}
