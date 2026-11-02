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
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/value_objects"
	db "github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure/persistence/postgres/generated"
)

type PaymentRepository struct {
	pool *pgxpool.Pool
}

func NewPaymentRepository(pool *pgxpool.Pool) (*PaymentRepository, error) {
	if pool == nil {
		return nil, fmt.Errorf("payment repository requires a database pool")
	}
	return &PaymentRepository{pool: pool}, nil
}

func (r *PaymentRepository) GetByID(
	ctx context.Context,
	id string,
) (*entities.Payment, error) {
	parsedID, err := parseUUID(id)
	if err != nil {
		return nil, err
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return nil, err
	}

	row, err := q.GetPaymentByID(ctx, parsedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get payment by ID: %w", err)
	}

	return mapPayment(row)
}

func (r *PaymentRepository) GetByBusinessReference(
	ctx context.Context,
	reference value_objects.BusinessReference,
) (*entities.Payment, error) {
	if reference.IsZero() {
		return nil, fmt.Errorf("business reference is required")
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return nil, err
	}

	row, err := q.GetPaymentByBusinessReference(
		ctx,
		db.GetPaymentByBusinessReferenceParams{
			BusinessReferenceType: reference.Type(),
			BusinessReferenceID:   reference.ID(),
		},
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get payment by business reference: %w", err)
	}

	return mapPayment(row)
}

func (r *PaymentRepository) ListByPayerID(
	ctx context.Context,
	payerID string,
	limit, offset int,
) ([]*entities.Payment, int64, error) {
	parsedID, err := parseUUID(payerID)
	if err != nil {
		return nil, 0, err
	}
	if limit <= 0 || offset < 0 {
		return nil, 0, fmt.Errorf("invalid payment pagination")
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return nil, 0, err
	}

	count, err := q.CountPaymentsByPayerID(ctx, parsedID)
	if err != nil {
		return nil, 0, fmt.Errorf("count payer payments: %w", err)
	}

	rows, err := q.ListPaymentsByPayerID(
		ctx,
		db.ListPaymentsByPayerIDParams{
			PayerID: parsedID,
			Limit:   int32(limit),
			Offset:  int32(offset),
		},
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list payer payments: %w", err)
	}

	result := make([]*entities.Payment, 0, len(rows))
	for _, row := range rows {
		payment, err := mapPayment(row)
		if err != nil {
			return nil, 0, fmt.Errorf("map listed payment: %w", err)
		}
		result = append(result, payment)
	}

	return result, count, nil
}

func (r *PaymentRepository) Save(
	ctx context.Context,
	payment *entities.Payment,
) error {
	if payment == nil {
		return fmt.Errorf("payment is required")
	}

	id, err := parseUUID(payment.ID())
	if err != nil {
		return err
	}
	payerID, err := parseUUID(payment.PayerID())
	if err != nil {
		return err
	}
	attemptID, err := nullableUUID(payment.AttemptID())
	if err != nil {
		return err
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return err
	}

	existing, err := q.GetPaymentByID(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		err = q.CreatePayment(ctx, db.CreatePaymentParams{
			ID:                    id,
			AttemptID:             attemptID,
			Reference:             payment.Reference(),
			PayerID:               payerID,
			Purpose:               string(payment.Purpose()),
			BusinessReferenceType: payment.BusinessReference().Type(),
			BusinessReferenceID:   payment.BusinessReference().ID(),
			Amount:                payment.Amount().Amount(),
			Currency:              payment.Amount().Currency(),
			Status:                string(payment.Status()),
			CreatedAt:             pgtype.Timestamptz{Time: payment.CreatedAt(), Valid: true},
			UpdatedAt:             pgtype.Timestamptz{Time: payment.UpdatedAt(), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("create payment: %w", err)
		}
		return nil

	case err != nil:
		return fmt.Errorf("check payment before save: %w", err)

	default:
		_ = existing
		err = q.UpdatePayment(ctx, db.UpdatePaymentParams{
			ID:        id,
			AttemptID: attemptID,
			Status:    string(payment.Status()),
			UpdatedAt: pgtype.Timestamptz{Time: payment.UpdatedAt(), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("update payment: %w", err)
		}
		return nil
	}
}

func mapPayment(row db.Payment) (*entities.Payment, error) {
	amount, err := value_objects.NewMoney(row.Amount, row.Currency)
	if err != nil {
		return nil, fmt.Errorf("reconstitute payment money: %w", err)
	}

	reference, err := value_objects.NewBusinessReference(
		row.BusinessReferenceType,
		row.BusinessReferenceID,
	)
	if err != nil {
		return nil, fmt.Errorf("reconstitute payment business reference: %w", err)
	}

	payment, err := entities.ReconstitutePayment(
		uuidString(row.ID),
		uuidString(row.AttemptID),
		row.Reference,
		uuidString(row.PayerID),
		enums.PaymentPurpose(row.Purpose),
		reference,
		amount,
		enums.PaymentStatus(row.Status),
		row.CreatedAt.Time,
		row.UpdatedAt.Time,
	)
	if err != nil {
		return nil, fmt.Errorf("reconstitute payment: %w", err)
	}

	return payment, nil
}

var _ ports.PaymentRepository = (*PaymentRepository)(nil)
