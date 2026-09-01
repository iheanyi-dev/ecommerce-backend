package security

import (
	"context"
	"errors"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"golang.org/x/crypto/bcrypt"
)

var ErrEmptyPassword = errors.New("password cannot be empty")

// BcryptPasswordHasher implements the application PasswordHasher port.
//
// The application layer depends only on ports.PasswordHasher and therefore
// has no knowledge of bcrypt or any other password hashing algorithm.
//
// This concrete implementation belongs to infrastructure because bcrypt is
// a technical/security implementation detail.
type BcryptPasswordHasher struct{}

// NewBcryptPasswordHasher creates a bcrypt-based password hasher.
func NewBcryptPasswordHasher() *BcryptPasswordHasher {
	return &BcryptPasswordHasher{}
}

// Hash securely hashes a plaintext password.
//
// The context is accepted because it is part of the application port.
// bcrypt itself does not currently provide context-aware hashing.
func (h *BcryptPasswordHasher) Hash(
	ctx context.Context,
	plainPassword string,
) (string, error) {
	// Respect an already-cancelled request before doing the expensive
	// password hashing operation.
	if err := ctx.Err(); err != nil {
		return "", err
	}

	if plainPassword == "" {
		return "", ErrEmptyPassword
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(plainPassword),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

// Compile-time assertion.
//
// If BcryptPasswordHasher ever stops implementing PasswordHasher, the
// compiler will immediately report the problem.
var _ ports.PasswordHasher = (*BcryptPasswordHasher)(nil)
