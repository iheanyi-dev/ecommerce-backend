package ports

import (
	"context"
	"time"
)

type RefreshTokenRecord struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

type RefreshTokenRepository interface {
	Create(
		ctx context.Context,
		record RefreshTokenRecord,
	) error

	FindByTokenHash(
		ctx context.Context,
		tokenHash string,
	) (*RefreshTokenRecord, error)

	Revoke(
		ctx context.Context,
		id string,
		revokedAt time.Time,
	) error

	// Rotate atomically revokes the existing refresh-token session and
	// creates its replacement.
	//
	// Both operations occur inside the same database transaction.
	// If either operation fails, neither change is committed.
	Rotate(
		ctx context.Context,
		oldTokenID string,
		revokedAt time.Time,
		newRecord RefreshTokenRecord,
	) error
}
