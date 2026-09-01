package infrastructure_test

import (
	"context"
	"testing"
	"errors"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/security"
	"golang.org/x/crypto/bcrypt"
)

func TestBcryptPasswordHasher_Hash(t *testing.T) {
	hasher := security.NewBcryptPasswordHasher()

	password := "SecurePassword123"

	hash, err := hasher.Hash(context.Background(), password)
	if err != nil {
		t.Fatalf("expected password hashing to succeed, got error: %v", err)
	}

	if hash == "" {
		t.Fatal("expected password hash to be returned")
	}

	if hash == password {
		t.Fatal("password must not be stored as plaintext")
	}

	// Verify that the generated hash can actually authenticate the original
	// password.
	if err := bcrypt.CompareHashAndPassword(
		[]byte(hash),
		[]byte(password),
	); err != nil {
		t.Fatalf(
			"expected generated hash to match original password: %v",
			err,
		)
	}
}

func TestBcryptPasswordHasher_RejectsEmptyPassword(t *testing.T) {
	hasher := security.NewBcryptPasswordHasher()

	_, err := hasher.Hash(
		context.Background(),
		"",
	)

	if err == nil {
		t.Fatal("expected empty password to return an error")
	}

	if !errors.Is(err, security.ErrEmptyPassword) {
		t.Fatalf(
			"expected ErrEmptyPassword, got %v",
			err,
		)
	}
}

func TestBcryptPasswordHasher_RespectsCancelledContext(t *testing.T) {
	hasher := security.NewBcryptPasswordHasher()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := hasher.Hash(
		ctx,
		"SecurePassword123",
	)

	if err == nil {
		t.Fatal("expected cancelled context to return an error")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"expected context.Canceled, got %v",
			err,
		)
	}
}