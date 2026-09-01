package ports

import "context"

// PasswordHasher defines the application boundary for password hashing.
//
// The Identity domain stores only PasswordHash values and has no knowledge
// of bcrypt, Argon2, or any other hashing algorithm. The infrastructure
// layer will provide the concrete implementation of this interface.
type PasswordHasher interface {
	Hash(ctx context.Context, plainPassword string) (string, error)
}
