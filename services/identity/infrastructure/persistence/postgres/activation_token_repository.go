package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/activation"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
	generated "github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/persistence/postgres/generated"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ActivationTokenRepository implements activation-token persistence using
// PostgreSQL. Account activation and token consumption are transactional.
type ActivationTokenRepository struct {
	pool    *pgxpool.Pool
	queries *generated.Queries
}

// NewActivationTokenRepository creates an activation-token repository.
func NewActivationTokenRepository(
	pool *pgxpool.Pool,
	queries *generated.Queries,
) *ActivationTokenRepository {
	return &ActivationTokenRepository{
		pool:    pool,
		queries: queries,
	}
}

// Create persists an activation token. Only the token hash is stored.
func (r *ActivationTokenRepository) Create(
	ctx context.Context,
	token *activation.Token,
) error {
	err := r.queries.CreateActivationToken(
		ctx,
		generated.CreateActivationTokenParams{
			ID:         pgUUID(token.ID()),
			UserID:     pgUUID(token.UserID().Value()),
			TokenHash:  token.TokenHash(),
			ExpiresAt:  pgTimestamp(token.ExpiresAt()),
			ConsumedAt: pgNullableTimestamp(token.ConsumedAt()),
			CreatedAt:  pgTimestamp(token.CreatedAt()),
		},
	)
	if err != nil {
		return fmt.Errorf(
			"create activation token: %w",
			application_errors.ErrActivationTokenPersistence,
		)
	}

	return nil
}

// DeleteByUserID invalidates all activation tokens belonging to a user.
func (r *ActivationTokenRepository) DeleteByUserID(
	ctx context.Context,
	userID user.UserID,
) error {
	if err := r.queries.DeleteActivationTokensByUserID(
		ctx,
		pgUUID(userID.Value()),
	); err != nil {
		return fmt.Errorf(
			"delete activation tokens: %w",
			application_errors.ErrActivationTokenPersistence,
		)
	}

	return nil
}

// ConsumeAndActivate atomically consumes a valid activation token and
// activates its pending user. Either both changes commit or neither does.
func (r *ActivationTokenRepository) ConsumeAndActivate(
	ctx context.Context,
	tokenHash string,
	at time.Time,
) error {
	at = at.UTC()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"begin activation transaction: %w",
			application_errors.ErrActivationTokenPersistence,
		)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txQueries := r.queries.WithTx(tx)

	token, err := txQueries.FindActivationTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return application_errors.ErrInvalidActivationToken
		}

		return fmt.Errorf(
			"find activation token: %w",
			application_errors.ErrActivationTokenPersistence,
		)
	}

	if token.ConsumedAt.Valid || !token.ExpiresAt.Time.After(at) {
		return application_errors.ErrInvalidActivationToken
	}

	consumed, err := txQueries.ConsumeActivationToken(
		ctx,
		generated.ConsumeActivationTokenParams{
			TokenHash:  tokenHash,
			ConsumedAt: pgTimestamp(at),
		},
	)
	if err != nil {
		return fmt.Errorf(
			"consume activation token: %w",
			application_errors.ErrActivationTokenPersistence,
		)
	}

	if consumed != 1 {
		return application_errors.ErrInvalidActivationToken
	}

	activated, err := txQueries.ActivatePendingUser(
		ctx,
		generated.ActivatePendingUserParams{
			ID:        token.UserID,
			UpdatedAt: pgTimestamp(at),
		},
	)
	if err != nil {
		return fmt.Errorf(
			"activate pending user: %w",
			application_errors.ErrActivationTokenPersistence,
		)
	}

	if activated != 1 {
		return application_errors.ErrInvalidActivationToken
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"commit activation transaction: %w",
			application_errors.ErrActivationTokenPersistence,
		)
	}

	return nil
}

var _ ports.ActivationTokenRepository = (*ActivationTokenRepository)(nil)
