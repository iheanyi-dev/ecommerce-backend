package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	apperrors "github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/valueobjects"
	db "github.com/iheanyi-dev/ecommerce-backend/services/promotion/infrastructure/persistence/postgres/generated"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PromotionRepository struct {
	pool *pgxpool.Pool
}

func NewPromotionRepository(pool *pgxpool.Pool) (*PromotionRepository, error) {
	if pool == nil {
		return nil, fmt.Errorf("promotion repository requires a database pool")
	}
	return &PromotionRepository{pool: pool}, nil
}

func (r *PromotionRepository) GetByID(
	ctx context.Context,
	promotionID string,
) (*entities.Promotion, error) {
	id, err := parseUUID(promotionID)
	if err != nil {
		return nil, err
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return nil, err
	}

	row, err := q.GetPromotionByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get promotion by ID: %w", err)
	}

	return mapPromotion(row)
}

func (r *PromotionRepository) CreateIfNoUnexpiredConflict(
	ctx context.Context,
	promotion *entities.Promotion,
	now time.Time,
) error {
	if promotion == nil {
		return fmt.Errorf("promotion is required")
	}
	if now.IsZero() {
		return fmt.Errorf("promotion conflict-check time is required")
	}

	id, err := parseUUID(promotion.ID())
	if err != nil {
		return err
	}
	storeID, err := parseUUID(promotion.StoreID())
	if err != nil {
		return err
	}
	productID, err := nullableUUID(promotion.ProductID())
	if err != nil {
		return err
	}
	customerID, err := nullableUUID(promotion.CustomerID())
	if err != nil {
		return err
	}

	percentage := pgtype.Int8{}
	if value, ok := promotion.Discount().PercentageBasisPoints(); ok {
		percentage = pgtype.Int8{Int64: value, Valid: true}
	}

	fixedAmount := pgtype.Int8{}
	fixedCurrency := pgtype.Text{}
	if value, ok := promotion.Discount().FixedAmount(); ok {
		fixedAmount = pgtype.Int8{Int64: value.Amount(), Valid: true}
		fixedCurrency = pgtype.Text{String: value.Currency(), Valid: true}
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return err
	}

	rows, err := q.CreatePromotionIfNoConflict(ctx, db.CreatePromotionIfNoConflictParams{
		ID:                    id,
		StoreID:               storeID,
		ProductID:             productID,
		CustomerID:            customerID,
		PercentageBasisPoints: percentage,
		FixedAmount:           fixedAmount,
		FixedCurrency:         fixedCurrency,
		StartsAt:              pgtype.Timestamptz{Time: promotion.StartsAt(), Valid: true},
		EndsAt:                pgtype.Timestamptz{Time: promotion.EndsAt(), Valid: true},
		Status:                string(promotion.StatusAt(now)),
		CreatedAt:             pgtype.Timestamptz{Time: promotion.CreatedAt(), Valid: true},
		UpdatedAt:             pgtype.Timestamptz{Time: promotion.UpdatedAt(), Valid: true},
		Now:                   pgtype.Timestamptz{Time: now.UTC(), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("create promotion if no unexpired conflict: %w", err)
	}
	if rows == 0 {
		return apperrors.ErrPromotionConflict
	}
	return nil
}

func (r *PromotionRepository) Update(
	ctx context.Context,
	promotion *entities.Promotion,
) error {
	if promotion == nil {
		return fmt.Errorf("promotion is required")
	}

	id, err := parseUUID(promotion.ID())
	if err != nil {
		return err
	}

	percentage := pgtype.Int8{}
	if value, ok := promotion.Discount().PercentageBasisPoints(); ok {
		percentage = pgtype.Int8{Int64: value, Valid: true}
	}

	fixedAmount := pgtype.Int8{}
	fixedCurrency := pgtype.Text{}
	if value, ok := promotion.Discount().FixedAmount(); ok {
		fixedAmount = pgtype.Int8{Int64: value.Amount(), Valid: true}
		fixedCurrency = pgtype.Text{String: value.Currency(), Valid: true}
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return err
	}

	rows, err := q.UpdatePromotion(ctx, db.UpdatePromotionParams{
		PercentageBasisPoints: percentage,
		FixedAmount:           fixedAmount,
		FixedCurrency:         fixedCurrency,
		StartsAt:              pgtype.Timestamptz{Time: promotion.StartsAt(), Valid: true},
		EndsAt:                pgtype.Timestamptz{Time: promotion.EndsAt(), Valid: true},
		Status:                string(promotion.StatusAt(promotion.UpdatedAt())),
		UpdatedAt:             pgtype.Timestamptz{Time: promotion.UpdatedAt(), Valid: true},
		ID:                    id,
	})
	if err != nil {
		return fmt.Errorf("update promotion: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("update promotion: promotion not found")
	}
	return nil
}

func (r *PromotionRepository) Delete(
	ctx context.Context,
	promotionID string,
) error {
	id, err := parseUUID(promotionID)
	if err != nil {
		return err
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return err
	}

	rows, err := q.DeletePromotion(ctx, id)
	if err != nil {
		return fmt.Errorf("delete promotion: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("delete promotion: promotion not found")
	}
	return nil
}

func (r *PromotionRepository) ListByStore(
	ctx context.Context,
	storeID string,
	limit, offset int,
) ([]*entities.Promotion, error) {
	id, err := parseUUID(storeID)
	if err != nil {
		return nil, err
	}
	if err := validatePagination(limit, offset); err != nil {
		return nil, err
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return nil, err
	}

	rows, err := q.ListPromotionsByStore(ctx, db.ListPromotionsByStoreParams{
		StoreID: id,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list promotions by store: %w", err)
	}

	return mapPromotions(rows)
}

func (r *PromotionRepository) ListActiveForTarget(
	ctx context.Context,
	storeID, productID, customerID string,
	now time.Time,
) ([]*entities.Promotion, error) {
	store, err := parseUUID(storeID)
	if err != nil {
		return nil, err
	}
	product, err := parseUUID(productID)
	if err != nil {
		return nil, err
	}
	customer, err := parseUUID(customerID)
	if err != nil {
		return nil, err
	}
	if now.IsZero() {
		return nil, fmt.Errorf("promotion query time is required")
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return nil, err
	}

	rows, err := q.ListActivePromotionsForTarget(
		ctx,
		db.ListActivePromotionsForTargetParams{
			StoreID:    store,
			Now:        pgtype.Timestamptz{Time: now.UTC(), Valid: true},
			ProductID:  product,
			CustomerID: customer,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list active promotions for target: %w", err)
	}

	return mapPromotions(rows)
}

func (r *PromotionRepository) ListAvailableForCustomer(
	ctx context.Context,
	customerID string,
	now time.Time,
	limit, offset int,
) ([]*entities.Promotion, error) {
	customer, err := parseUUID(customerID)
	if err != nil {
		return nil, err
	}
	if now.IsZero() {
		return nil, fmt.Errorf("promotion query time is required")
	}
	if err := validatePagination(limit, offset); err != nil {
		return nil, err
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return nil, err
	}

	rows, err := q.ListAvailablePromotionsForCustomer(
		ctx,
		db.ListAvailablePromotionsForCustomerParams{
			Now:         pgtype.Timestamptz{Time: now.UTC(), Valid: true},
			CustomerID:  customer,
			LimitCount:  int32(limit),
			OffsetCount: int32(offset),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list available promotions for customer: %w", err)
	}

	return mapPromotions(rows)
}

func (r *PromotionRepository) ListScheduledDue(
	ctx context.Context,
	now time.Time,
	limit int,
) ([]string, error) {
	return r.listDueIDs(ctx, now, limit, true)
}

func (r *PromotionRepository) ListExpiredDue(
	ctx context.Context,
	now time.Time,
	limit int,
) ([]string, error) {
	return r.listDueIDs(ctx, now, limit, false)
}

func (r *PromotionRepository) listDueIDs(
	ctx context.Context,
	now time.Time,
	limit int,
	scheduled bool,
) ([]string, error) {
	if now.IsZero() {
		return nil, fmt.Errorf("promotion query time is required")
	}
	if err := validatePagination(limit, 0); err != nil {
		return nil, err
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return nil, err
	}

	queryTime := pgtype.Timestamptz{Time: now.UTC(), Valid: true}
	batchSize := int32(limit)

	var ids []pgtype.UUID
	if scheduled {
		ids, err = q.ListScheduledPromotionsDue(ctx, db.ListScheduledPromotionsDueParams{
			Now: queryTime, BatchSize: batchSize,
		})
	} else {
		ids, err = q.ListExpiredPromotionsDue(ctx, db.ListExpiredPromotionsDueParams{
			Now: queryTime, BatchSize: batchSize,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("list due promotion IDs: %w", err)
	}

	result := make([]string, 0, len(ids))
	for _, id := range ids {
		result = append(result, uuidString(id))
	}
	return result, nil
}

func mapPromotions(rows []db.Promotion) ([]*entities.Promotion, error) {
	result := make([]*entities.Promotion, 0, len(rows))
	for _, row := range rows {
		promotion, err := mapPromotion(row)
		if err != nil {
			return nil, fmt.Errorf("map promotion row: %w", err)
		}
		result = append(result, promotion)
	}
	return result, nil
}

func mapPromotion(row db.Promotion) (*entities.Promotion, error) {
	var percentage *int64
	if row.PercentageBasisPoints.Valid {
		value := row.PercentageBasisPoints.Int64
		percentage = &value
	}

	var fixedAmount *valueobjects.Money
	if row.FixedAmount.Valid || row.FixedCurrency.Valid {
		if !row.FixedAmount.Valid || !row.FixedCurrency.Valid {
			return nil, fmt.Errorf("persisted promotion has incomplete fixed discount")
		}
		money, err := valueobjects.NewMoney(
			row.FixedAmount.Int64,
			row.FixedCurrency.String,
		)
		if err != nil {
			return nil, fmt.Errorf("reconstitute promotion fixed amount: %w", err)
		}
		fixedAmount = &money
	}

	discount, err := valueobjects.NewDiscount(percentage, fixedAmount)
	if err != nil {
		return nil, fmt.Errorf("reconstitute promotion discount: %w", err)
	}

	promotion, err := entities.ReconstitutePromotion(
		uuidString(row.ID),
		uuidString(row.StoreID),
		nullableString(row.ProductID),
		nullableString(row.CustomerID),
		discount,
		row.StartsAt.Time,
		row.EndsAt.Time,
		row.CreatedAt.Time,
		row.UpdatedAt.Time,
		entities.PromotionStatus(row.Status),
	)
	if err != nil {
		return nil, fmt.Errorf("reconstitute promotion: %w", err)
	}

	return promotion, nil
}

func validatePagination(limit, offset int) error {
	if limit <= 0 || offset < 0 ||
		int64(limit) > math.MaxInt32 || int64(offset) > math.MaxInt32 {
		return fmt.Errorf("invalid promotion pagination")
	}
	return nil
}

var _ ports.PromotionRepository = (*PromotionRepository)(nil)
