package postgres

import (
	"context"
	"encoding/json"
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

type PaymentAttemptRepository struct {
	pool *pgxpool.Pool
}

func NewPaymentAttemptRepository(pool *pgxpool.Pool) (*PaymentAttemptRepository, error) {
	if pool == nil {
		return nil, fmt.Errorf("payment attempt repository requires a database pool")
	}
	return &PaymentAttemptRepository{pool: pool}, nil
}

func (r *PaymentAttemptRepository) GetByID(
	ctx context.Context,
	id string,
) (*entities.PaymentAttempt, error) {
	parsedID, err := parseUUID(id)
	if err != nil {
		return nil, err
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return nil, err
	}

	row, err := q.GetPaymentAttemptByID(ctx, parsedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get payment attempt by ID: %w", err)
	}

	return mapPaymentAttempt(row)
}

func (r *PaymentAttemptRepository) Save(
	ctx context.Context,
	attempt *entities.PaymentAttempt,
) error {
	if attempt == nil {
		return fmt.Errorf("payment attempt is required")
	}

	id, err := parseUUID(attempt.ID())
	if err != nil {
		return err
	}
	paymentID, err := parseUUID(attempt.PaymentID())
	if err != nil {
		return err
	}

	metadata := attempt.Metadata()
	if len(metadata) == 0 {
		metadata = json.RawMessage("{}")
	}

	var expiresAt pgtype.Timestamptz
	if value := attempt.ExpiresAt(); value != nil {
		expiresAt = pgtype.Timestamptz{Time: *value, Valid: true}
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return err
	}

	_, err = q.GetPaymentAttemptByID(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		err = q.CreatePaymentAttempt(ctx, db.CreatePaymentAttemptParams{
			ID:                id,
			PaymentID:         paymentID,
			Provider:          attempt.Provider(),
			ProviderReference: nullableText(attempt.ProviderReference()),
			CustomerEmail:     nullableText(attempt.CustomerEmail()),
			CheckoutUrl:       nullableText(attempt.CheckoutURL()),
			ExpiresAt:         expiresAt,
			Metadata:          []byte(metadata),
			Status:            string(attempt.Status()),
			FailureReason:     attempt.FailureReason(),
			CreatedAt:         pgtype.Timestamptz{Time: attempt.CreatedAt(), Valid: true},
			UpdatedAt:         pgtype.Timestamptz{Time: attempt.UpdatedAt(), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("create payment attempt: %w", err)
		}
		return nil

	case err != nil:
		return fmt.Errorf("check payment attempt before save: %w", err)

	default:
		err = q.UpdatePaymentAttempt(ctx, db.UpdatePaymentAttemptParams{
			ID:                id,
			ProviderReference: nullableText(attempt.ProviderReference()),
			CustomerEmail:     nullableText(attempt.CustomerEmail()),
			CheckoutUrl:       nullableText(attempt.CheckoutURL()),
			ExpiresAt:         expiresAt,
			Metadata:          []byte(metadata),
			Status:            string(attempt.Status()),
			FailureReason:     attempt.FailureReason(),
			UpdatedAt:         pgtype.Timestamptz{Time: attempt.UpdatedAt(), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("update payment attempt: %w", err)
		}
		return nil
	}
}

func mapPaymentAttempt(row db.PaymentAttempt) (*entities.PaymentAttempt, error) {
	attempt, err := entities.ReconstitutePaymentAttempt(
		uuidString(row.ID),
		uuidString(row.PaymentID),
		row.Provider,
		row.ProviderReference.String,
		row.CustomerEmail.String,
		row.CheckoutUrl.String,
		nullableTime(row.ExpiresAt),
		json.RawMessage(row.Metadata),
		enums.PaymentAttemptStatus(row.Status),
		row.FailureReason,
		row.CreatedAt.Time,
		row.UpdatedAt.Time,
	)
	if err != nil {
		return nil, fmt.Errorf("reconstitute payment attempt: %w", err)
	}
	return attempt, nil
}

var _ ports.PaymentAttemptRepository = (*PaymentAttemptRepository)(nil)
