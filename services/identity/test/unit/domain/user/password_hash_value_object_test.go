package user_test

import (
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

func TestNewPasswordHash_WithValidHash_CreatesPasswordHash(t *testing.T) {
	t.Parallel()

	const hash = "$argon2id$v=19$m=65536,t=3,p=2$example"

	passwordHash, err := user.NewPasswordHash(hash)

	if err != nil {
		t.Fatalf("expected password hash creation to succeed, got error: %v", err)
	}

	if passwordHash.String() != hash {
		t.Fatalf(
			"expected hash %q, got %q",
			hash,
			passwordHash.String(),
		)
	}
}

func TestNewPasswordHash_WithEmptyHash_ReturnsError(t *testing.T) {
	t.Parallel()

	_, err := user.NewPasswordHash("")

	if err == nil {
		t.Fatal("expected empty password hash to be rejected")
	}
}
