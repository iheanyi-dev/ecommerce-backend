package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/value_objects"
	db "github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure/persistence/postgres/generated"
)

type PayoutRepository struct {
	pool      *pgxpool.Pool
	encryptor *BeneficiaryEncryptor
}

func NewPayoutRepository(
	pool *pgxpool.Pool,
	encryptor *BeneficiaryEncryptor,
) (*PayoutRepository, error) {
	if pool == nil {
		return nil, fmt.Errorf("payout repository requires a database pool")
	}
	if encryptor == nil {
		return nil, fmt.Errorf("payout repository requires a beneficiary encryptor")
	}
	return &PayoutRepository{pool: pool, encryptor: encryptor}, nil
}

func (r *PayoutRepository) GetByID(
	ctx context.Context,
	id string,
) (*entities.Payout, error) {
	parsedID, err := parseUUID(id)
	if err != nil {
		return nil, err
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return nil, err
	}

	row, err := q.GetPayoutByID(ctx, parsedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get payout by ID: %w", err)
	}

	return r.mapPayout(row)
}

func (r *PayoutRepository) GetByBusinessReference(
	ctx context.Context,
	reference value_objects.BusinessReference,
) (*entities.Payout, error) {
	if reference.IsZero() {
		return nil, fmt.Errorf("payout business reference is required")
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return nil, err
	}

	row, err := q.GetPayoutByBusinessReference(
		ctx,
		db.GetPayoutByBusinessReferenceParams{
			BusinessReferenceType: nullableText(reference.Type()),
			BusinessReferenceID:   nullableText(reference.ID()),
		},
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get payout by business reference: %w", err)
	}

	return r.mapPayout(row)
}

func (r *PayoutRepository) ListByStoreID(
	ctx context.Context,
	storeID string,
	limit, offset int,
) ([]*entities.Payout, int64, error) {
	parsedStoreID, err := parseUUID(storeID)
	if err != nil {
		return nil, 0, err
	}
	if limit <= 0 || offset < 0 || int64(limit) > math.MaxInt32 || int64(offset) > math.MaxInt32 {
		return nil, 0, fmt.Errorf("invalid payout pagination")
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return nil, 0, err
	}

	total, err := q.CountPayoutsByStoreID(ctx, parsedStoreID)
	if err != nil {
		return nil, 0, fmt.Errorf("count store payouts: %w", err)
	}

	rows, err := q.ListPayoutsByStoreID(ctx, db.ListPayoutsByStoreIDParams{
		StoreID: parsedStoreID,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list store payouts: %w", err)
	}

	result := make([]*entities.Payout, 0, len(rows))
	for _, row := range rows {
		payout, err := r.mapPayout(row)
		if err != nil {
			return nil, 0, fmt.Errorf("map listed payout: %w", err)
		}
		result = append(result, payout)
	}

	return result, total, nil
}

func (r *PayoutRepository) Save(
	ctx context.Context,
	payout *entities.Payout,
) error {
	if payout == nil {
		return fmt.Errorf("payout is required")
	}

	id, err := parseUUID(payout.ID())
	if err != nil {
		return err
	}
	attemptID, err := nullableUUID(payout.AttemptID())
	if err != nil {
		return err
	}
	storeID, err := parseUUID(payout.StoreID())
	if err != nil {
		return err
	}
	orderID, err := nullableUUID(payout.OrderID())
	if err != nil {
		return err
	}

	encryptedBeneficiary, err := r.encryptor.Encrypt(payout.Beneficiary())
	if err != nil {
		return fmt.Errorf("encrypt payout beneficiary: %w", err)
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return err
	}

	_, err = q.GetPayoutByID(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		reference := payout.BusinessReference()
		err = q.CreatePayout(ctx, db.CreatePayoutParams{
			ID:                           id,
			AttemptID:                    attemptID,
			Reference:                    payout.Reference(),
			BusinessReferenceType:        nullableText(reference.Type()),
			BusinessReferenceID:          nullableText(reference.ID()),
			StoreID:                      storeID,
			OrderID:                      orderID,
			Reason:                       payout.Reason(),
			BeneficiarySnapshotEncrypted: encryptedBeneficiary,
			Amount:                       payout.Amount().Amount(),
			Currency:                     payout.Amount().Currency(),
			Status:                       string(payout.Status()),
			CreatedAt:                    pgtype.Timestamptz{Time: payout.CreatedAt(), Valid: true},
			UpdatedAt:                    pgtype.Timestamptz{Time: payout.UpdatedAt(), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("create payout: %w", err)
		}
		return nil

	case err != nil:
		return fmt.Errorf("check payout before save: %w", err)

	default:
		err = q.UpdatePayout(ctx, db.UpdatePayoutParams{
			ID:        id,
			AttemptID: attemptID,
			Status:    string(payout.Status()),
			UpdatedAt: pgtype.Timestamptz{Time: payout.UpdatedAt(), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("update payout: %w", err)
		}
		return nil
	}
}

func (r *PayoutRepository) mapPayout(row db.Payout) (*entities.Payout, error) {
	beneficiary, err := r.encryptor.Decrypt(row.BeneficiarySnapshotEncrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt payout beneficiary: %w", err)
	}

	amount, err := value_objects.NewMoney(row.Amount, row.Currency)
	if err != nil {
		return nil, fmt.Errorf("reconstitute payout amount: %w", err)
	}

	var reference value_objects.BusinessReference
	if row.BusinessReferenceType.Valid != row.BusinessReferenceID.Valid {
		return nil, fmt.Errorf("payout has an incomplete business reference")
	}
	if row.BusinessReferenceType.Valid {
		reference, err = value_objects.NewBusinessReference(
			row.BusinessReferenceType.String,
			row.BusinessReferenceID.String,
		)
		if err != nil {
			return nil, fmt.Errorf("reconstitute payout business reference: %w", err)
		}
	}

	payout, err := entities.ReconstitutePayout(
		uuidString(row.ID),
		uuidString(row.AttemptID),
		row.Reference,
		reference,
		row.Reason,
		beneficiary,
		uuidString(row.StoreID),
		uuidString(row.OrderID),
		amount,
		enums.PayoutStatus(row.Status),
		row.CreatedAt.Time,
		row.UpdatedAt.Time,
	)
	if err != nil {
		return nil, fmt.Errorf("reconstitute payout: %w", err)
	}

	return payout, nil
}

var _ ports.PayoutRepository = (*PayoutRepository)(nil)
