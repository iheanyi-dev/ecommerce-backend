package user_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

func TestNewUserID_GeneratesValidID(t *testing.T) {
	t.Parallel()

	id := user.NewUserID()

	if id.String() == "" {
		t.Fatal("expected generated user ID to have a value")
	}

	if _, err := uuid.Parse(id.String()); err != nil {
		t.Fatalf("expected valid UUID, got %q: %v", id.String(), err)
	}
}

func TestUserIDFromString_WithValidUUID_CreatesUserID(t *testing.T) {
	t.Parallel()

	rawID := uuid.New()

	id, err := user.UserIDFromString(rawID.String())
	if err != nil {
		t.Fatalf("expected valid UUID to succeed, got error: %v", err)
	}

	if id.String() != rawID.String() {
		t.Fatalf(
			"expected ID %q, got %q",
			rawID.String(),
			id.String(),
		)
	}
}

func TestUserIDFromString_WithInvalidUUID_ReturnsError(t *testing.T) {
	t.Parallel()

	_, err := user.UserIDFromString("not-a-uuid")

	if err == nil {
		t.Fatal("expected invalid UUID to return an error")
	}
}
