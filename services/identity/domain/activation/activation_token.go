package activation

import (
	"encoding/hex"
	"time"

	"github.com/google/uuid"

	domain_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// Token represents a single-use email activation token.
// Only the token hash is stored; the raw token is never part of the domain entity.
type Token struct {
	id         uuid.UUID
	userID     user.UserID
	tokenHash  string
	expiresAt  time.Time
	consumedAt *time.Time
	createdAt  time.Time
}

// NewToken creates an activation token for a user.
func NewToken(userID user.UserID, tokenHash string, expiresAt time.Time) (*Token, error) {
	if userID.Value() == uuid.Nil {
		return nil, domain_errors.ErrInvalidUserID
	}

	if len(tokenHash) != 64 {
		return nil, domain_errors.ErrInvalidActivationTokenHash
	}

	if _, err := hex.DecodeString(tokenHash); err != nil {
		return nil, domain_errors.ErrInvalidActivationTokenHash
	}

	now := time.Now().UTC()
	expiresAt = expiresAt.UTC()

	if !expiresAt.After(now) {
		return nil, domain_errors.ErrInvalidActivationTokenExpiry
	}

	return &Token{
		id:        uuid.New(),
		userID:    userID,
		tokenHash: tokenHash,
		expiresAt: expiresAt,
		createdAt: now,
	}, nil
}

// ID returns the token's unique identifier.
func (t *Token) ID() uuid.UUID {
	return t.id
}

// UserID returns the ID of the user associated with the token.
func (t *Token) UserID() user.UserID {
	return t.userID
}

// TokenHash returns the stored token hash.
func (t *Token) TokenHash() string {
	return t.tokenHash
}

// ExpiresAt returns the token expiry timestamp.
func (t *Token) ExpiresAt() time.Time {
	return t.expiresAt
}

// CreatedAt returns the token creation timestamp.
func (t *Token) CreatedAt() time.Time {
	return t.createdAt
}

// ConsumedAt returns when the token was consumed, if applicable.
func (t *Token) ConsumedAt() *time.Time {
	if t.consumedAt == nil {
		return nil
	}

	consumedAt := *t.consumedAt
	return &consumedAt
}

// IsExpired reports whether the token has expired at the supplied time.
func (t *Token) IsExpired(at time.Time) bool {
	return !at.Before(t.expiresAt)
}

// IsConsumed reports whether the token has already been consumed.
func (t *Token) IsConsumed() bool {
	return t.consumedAt != nil
}

// Consume marks the token as consumed.
func (t *Token) Consume(at time.Time) error {
	if t.IsConsumed() {
		return domain_errors.ErrActivationTokenAlreadyConsumed
	}

	at = at.UTC()

	if t.IsExpired(at) {
		return domain_errors.ErrActivationTokenExpired
	}

	t.consumedAt = &at
	return nil
}
