package ports

import "context"

// RefreshTokenService defines the application boundary for creating
// and protecting refresh tokens.
//
// The application layer deliberately does not know how refresh tokens
// are generated or hashed. Infrastructure provides the implementation.
type RefreshTokenService interface {
	// Generate creates a new cryptographically secure refresh token.
	//
	// The returned value is the raw token that may be sent to the client.
	// The raw token must never be persisted directly.
	Generate(ctx context.Context) (string, error)

	// Hash creates a one-way representation of a refresh token suitable
	// for persistent storage.
	Hash(ctx context.Context, token string) (string, error)
}
