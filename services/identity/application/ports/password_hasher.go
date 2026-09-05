package ports

import "context"

// PasswordHasher defines the application boundary for password hashing
// and password verification.
//
// The application layer does not depend on bcrypt or any other concrete
// hashing algorithm. Infrastructure provides the implementation.
type PasswordHasher interface {
	// Hash securely hashes a plaintext password.
	Hash(
		ctx context.Context,
		plainPassword string,
	) (string, error)

	// Verify checks whether a plaintext password matches a stored hash.
	//
	// The plaintext password is supplied by the user during authentication.
	// The stored hash comes from the persisted User aggregate.
	Verify(
		ctx context.Context,
		plainPassword string,
		passwordHash string,
	) error
}
