package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
	db "github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure/persistence/postgres/generated"
)

type PayoutAttemptRepository struct {
	pool *pgxpool.Pool
}

func NewPayoutAttemptRepository(pool *pgxpool.Pool) (*PayoutAttemptRepository, error) {
	if pool == nil {
		return nil, fmt.Errorf("payout attempt repository requires a database pool")
	}
	return &PayoutAttemptRepository{pool: pool}, nil
}

func (r *PayoutAttemptRepository) GetByID(
	ctx context.Context,
	id string,
) (*entities.PayoutAttempt, error) {
	parsedID, err := parseUUID(id)
	if err != nil {
		return nil, err
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return nil, err
	}

	row, err := q.GetPayoutAttemptByID(ctx, parsedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get payout attempt by ID: %w", err)
	}

	return mapPayoutAttempt(row)
}

func (r *PayoutAttemptRepository) GetByPayoutID(
	ctx context.Context,
	payoutID string,
) (*entities.PayoutAttempt, error) {
	parsedID, err := parseUUID(payoutID)
	if err != nil {
		return nil, err
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return nil, err
	}

	row, err := q.GetLatestPayoutAttemptByPayoutID(ctx, parsedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest payout attempt: %w", err)
	}

	return mapPayoutAttempt(row)
}

func (r *PayoutAttemptRepository) Save(
	ctx context.Context,
	attempt *entities.PayoutAttempt,
) error {
	if attempt == nil {
		return fmt.Errorf("payout attempt is required")
	}

	id, err := parseUUID(attempt.ID())
	if err != nil {
		return err
	}
	payoutID, err := parseUUID(attempt.PayoutID())
	if err != nil {
		return err
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return err
	}

	_, err = q.GetPayoutAttemptByID(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		err = q.CreatePayoutAttempt(ctx, db.CreatePayoutAttemptParams{
			ID:                id,
			PayoutID:          payoutID,
			Provider:          attempt.Provider(),
			ProviderReference: nullableText(attempt.ProviderReference()),
			Status:            string(attempt.Status()),
			FailureReason:     attempt.FailureReason(),
			CreatedAt:         pgtype.Timestamptz{Time: attempt.CreatedAt(), Valid: true},
			UpdatedAt:         pgtype.Timestamptz{Time: attempt.UpdatedAt(), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("create payout attempt: %w", err)
		}
		return nil

	case err != nil:
		return fmt.Errorf("check payout attempt before save: %w", err)

	default:
		err = q.UpdatePayoutAttempt(ctx, db.UpdatePayoutAttemptParams{
			ID:                id,
			ProviderReference: nullableText(attempt.ProviderReference()),
			Status:            string(attempt.Status()),
			FailureReason:     attempt.FailureReason(),
			UpdatedAt:         pgtype.Timestamptz{Time: attempt.UpdatedAt(), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("update payout attempt: %w", err)
		}
		return nil
	}
}

func mapPayoutAttempt(row db.PayoutAttempt) (*entities.PayoutAttempt, error) {
	attempt, err := entities.ReconstitutePayoutAttempt(
		uuidString(row.ID),
		uuidString(row.PayoutID),
		row.Provider,
		row.ProviderReference.String,
		enums.PayoutAttemptStatus(row.Status),
		row.FailureReason,
		row.CreatedAt.Time,
		row.UpdatedAt.Time,
	)
	if err != nil {
		return nil, fmt.Errorf("reconstitute payout attempt: %w", err)
	}
	return attempt, nil
}

var _ ports.PayoutAttemptRepository = (*PayoutAttemptRepository)(nil)
