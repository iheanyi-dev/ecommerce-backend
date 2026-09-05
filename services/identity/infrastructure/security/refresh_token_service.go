package security

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// RefreshTokenService implements the application-level
// RefreshTokenService port.
//
// Refresh tokens are generated using cryptographically secure random bytes.
// Only a one-way SHA-256 hash of the token is intended to be persisted.
//
// The raw refresh token is returned only to the caller so it can be delivered
// to the client. Infrastructure persistence receives only the hash.
type RefreshTokenService struct{}

// NewRefreshTokenService creates a new refresh-token service.
func NewRefreshTokenService() *RefreshTokenService {
	return &RefreshTokenService{}
}

// Generate creates a cryptographically secure refresh token.
//
// 32 random bytes provide 256 bits of entropy. The token is encoded using
// unpadded URL-safe Base64 so it can safely travel through JSON, HTTP headers,
// cookies, and other transport mechanisms.
func (s *RefreshTokenService) Generate(
	_ context.Context,
) (string, error) {
	const tokenSize = 32

	randomBytes := make([]byte, tokenSize)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate secure refresh token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(randomBytes), nil
}

// Hash creates the deterministic one-way hash used to identify a persisted
// refresh-token session.
//
// A deterministic hash is required because the refresh-token repository must
// be able to locate the stored session using the hash of the token supplied
// by the client.
//
// The raw refresh token must never be persisted.
func (s *RefreshTokenService) Hash(
	_ context.Context,
	token string,
) (string, error) {
	if token == "" {
		return "", fmt.Errorf("refresh token cannot be empty")
	}

	hash := sha256.Sum256([]byte(token))

	return base64.RawURLEncoding.EncodeToString(hash[:]), nil
}

var _ interface {
	Generate(context.Context) (string, error)
	Hash(context.Context, string) (string, error)
} = (*RefreshTokenService)(nil)
