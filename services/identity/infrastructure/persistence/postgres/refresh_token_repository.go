package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	generated "github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/persistence/postgres/generated"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RefreshTokenRepository implements the application-level
// RefreshTokenRepository port using PostgreSQL and SQLC.
//
// The application layer works with RefreshTokenRecord values and must not
// know anything about pgtype or SQLC-generated persistence models.
//
// This repository is therefore responsible for translating between the
// application representation and the PostgreSQL representation.
type RefreshTokenRepository struct {
	pool    *pgxpool.Pool
	queries *generated.Queries
}

// NewRefreshTokenRepository creates a refresh-token repository backed by
// the supplied SQLC query executor.
//
// The executor may be backed by either a PostgreSQL connection pool or a
// transaction, which allows the same repository implementation to be used
// in normal application execution and transaction-backed integration tests.
func NewRefreshTokenRepository(
	pool *pgxpool.Pool,
	queries *generated.Queries,
) *RefreshTokenRepository {
	return &RefreshTokenRepository{
		pool:    pool,
		queries: queries,
	}
}

// Create persists a new refresh-token session.
//
// The raw refresh token is never passed to this repository. The application
// layer provides only the cryptographic hash of the token.
func (r *RefreshTokenRepository) Create(
	ctx context.Context,
	record ports.RefreshTokenRecord,
) error {
	id, err := uuid.Parse(record.ID)
	if err != nil {
		return fmt.Errorf("parse refresh token ID: %w", err)
	}

	userID, err := uuid.Parse(record.UserID)
	if err != nil {
		return fmt.Errorf("parse refresh token user ID: %w", err)
	}

	params := generated.CreateRefreshTokenParams{
		ID:        pgUUID(id),
		UserID:    pgUUID(userID),
		TokenHash: record.TokenHash,
		ExpiresAt: pgTimestamp(record.ExpiresAt),
		RevokedAt: pgNullableTimestamp(record.RevokedAt),
		CreatedAt: pgTimestamp(record.CreatedAt),
	}

	if err := r.queries.CreateRefreshToken(ctx, params); err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}

	return nil
}

// FindByTokenHash retrieves a refresh-token session using the stored token
// hash.
//
// A missing token is represented by nil, nil at the application boundary.
// This allows the use case to distinguish "not found" from an actual
// persistence failure.
func (r *RefreshTokenRepository) FindByTokenHash(
	ctx context.Context,
	tokenHash string,
) (*ports.RefreshTokenRecord, error) {
	row, err := r.queries.FindRefreshTokenByHash(
		ctx,
		tokenHash,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf(
			"find refresh token by hash: %w",
			err,
		)
	}

	id := row.ID.String()
	if id == "" {
		return nil, fmt.Errorf(
			"refresh token record contains invalid ID",
		)
	}

	userID := row.UserID.String()
	if userID == "" {
		return nil, fmt.Errorf(
			"refresh token record contains invalid user ID",
		)
	}

	record := &ports.RefreshTokenRecord{
		ID:        id,
		UserID:    userID,
		TokenHash: row.TokenHash,
		ExpiresAt: row.ExpiresAt.Time,
		CreatedAt: row.CreatedAt.Time,
	}

	if row.RevokedAt.Valid {
		revokedAt := row.RevokedAt.Time
		record.RevokedAt = &revokedAt
	}

	return record, nil
}

// Revoke marks a refresh-token session as revoked.
//
// Revocation is deliberately represented by a timestamp rather than
// deleting the record. Keeping the record allows us to retain session
// history and makes token lifecycle auditing possible later.
func (r *RefreshTokenRepository) Revoke(
	ctx context.Context,
	id string,
	revokedAt time.Time,
) error {
	tokenID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("parse refresh token ID: %w", err)
	}

	params := generated.RevokeRefreshTokenParams{
		ID:        pgUUID(tokenID),
		RevokedAt: pgTimestamp(revokedAt),
	}

	if err := r.queries.RevokeRefreshToken(
		ctx,
		params,
	); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	return nil
}

// pgUUID converts a standard UUID into the pgtype representation expected
// by SQLC.
func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: id,
		Valid: true,
	}
}

// pgTimestamp converts a time.Time into the pgtype representation expected
// by SQLC.
func pgTimestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:  value,
		Valid: true,
	}
}

// pgNullableTimestamp converts an optional timestamp into its PostgreSQL
// nullable representation.
//
// A nil RevokedAt means that the refresh-token session is still active.
func pgNullableTimestamp(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{
			Valid: false,
		}
	}

	return pgtype.Timestamptz{
		Time:  *value,
		Valid: true,
	}
}

// Rotate atomically revokes the existing refresh-token session and creates
// its replacement.
//
// Both database operations execute inside the same PostgreSQL transaction.
//
// If revocation succeeds but creation fails, the transaction is rolled back,
// meaning the original refresh token remains valid. This prevents the system
// from leaving a user without a usable refresh session because of a partial
// persistence failure.
func (r *RefreshTokenRepository) Rotate(
	ctx context.Context,
	oldTokenID string,
	revokedAt time.Time,
	newRecord ports.RefreshTokenRecord,
) error {
	oldID, err := uuid.Parse(oldTokenID)
	if err != nil {
		return fmt.Errorf("parse old refresh token ID: %w", err)
	}

	newID, err := uuid.Parse(newRecord.ID)
	if err != nil {
		return fmt.Errorf("parse new refresh token ID: %w", err)
	}

	userID, err := uuid.Parse(newRecord.UserID)
	if err != nil {
		return fmt.Errorf("parse new refresh token user ID: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin refresh token rotation transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txQueries := r.queries.WithTx(tx)

	if err := txQueries.RevokeRefreshToken(
		ctx,
		generated.RevokeRefreshTokenParams{
			ID:        pgUUID(oldID),
			RevokedAt: pgTimestamp(revokedAt),
		},
	); err != nil {
		return fmt.Errorf("revoke old refresh token: %w", err)
	}

	if err := txQueries.CreateRefreshToken(
		ctx,
		generated.CreateRefreshTokenParams{
			ID:        pgUUID(newID),
			UserID:    pgUUID(userID),
			TokenHash: newRecord.TokenHash,
			ExpiresAt: pgTimestamp(newRecord.ExpiresAt),
			RevokedAt: pgNullableTimestamp(newRecord.RevokedAt),
			CreatedAt: pgTimestamp(newRecord.CreatedAt),
		},
	); err != nil {
		return fmt.Errorf("create replacement refresh token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit refresh token rotation: %w", err)
	}

	return nil
}

// Compile-time assertion.
//
// If the repository ever stops satisfying the application port, the
// compiler will immediately report the architectural violation.
var _ ports.RefreshTokenRepository = (*RefreshTokenRepository)(nil)
