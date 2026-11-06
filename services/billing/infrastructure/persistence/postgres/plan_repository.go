package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/value_objects"
	dbgen "github.com/iheanyi-dev/ecommerce-backend/services/billing/infrastructure/persistence/postgres/generated"
)

type PlanRepository struct {
	pool *pgxpool.Pool
}

func NewPlanRepository(pool *pgxpool.Pool) (*PlanRepository, error) {
	if pool == nil {
		return nil, fmt.Errorf("billing database pool is required")
	}
	return &PlanRepository{pool: pool}, nil
}

func (r *PlanRepository) Create(ctx context.Context, plan entities.Plan) error {
	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return err
	}
	promotionName, discount, starts, ends := planPromotionParams(plan.Promotion)
	err = q.CreatePlan(ctx, dbgen.CreatePlanParams{
		ID:   pgtype.UUID{Bytes: plan.ID, Valid: true},
		Type: plan.Type, Name: plan.Name, Description: plan.Description,
		PriceAmountMinor: plan.Price.AmountMinor, Currency: plan.Price.Currency,
		DurationNs: int64(plan.Duration), MaxProducts: int32(plan.Capabilities.MaxProducts),
		AiEnabled: plan.Capabilities.AIEnabled, Available: plan.Available,
		PromotionName: promotionName, PromotionDiscountAmountMinor: discount,
		PromotionStartsAt: starts, PromotionEndsAt: ends,
		CreatedAt: pgtype.Timestamptz{Time: plan.CreatedAt, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: plan.UpdatedAt, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("create plan: %w", err)
	}
	return nil
}

func (r *PlanRepository) Update(ctx context.Context, plan entities.Plan) error {
	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return err
	}
	promotionName, discount, starts, ends := planPromotionParams(plan.Promotion)
	err = q.UpdatePlan(ctx, dbgen.UpdatePlanParams{
		ID:   pgtype.UUID{Bytes: plan.ID, Valid: true},
		Type: plan.Type, Name: plan.Name, Description: plan.Description,
		PriceAmountMinor: plan.Price.AmountMinor, Currency: plan.Price.Currency,
		DurationNs: int64(plan.Duration), MaxProducts: int32(plan.Capabilities.MaxProducts),
		AiEnabled: plan.Capabilities.AIEnabled, Available: plan.Available,
		PromotionName: promotionName, PromotionDiscountAmountMinor: discount,
		PromotionStartsAt: starts, PromotionEndsAt: ends,
		UpdatedAt: pgtype.Timestamptz{Time: plan.UpdatedAt, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("update plan: %w", err)
	}
	return nil
}

func (r *PlanRepository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return err
	}
	if err := q.DeletePlanByID(ctx, pgtype.UUID{Bytes: id, Valid: true}); err != nil {
		return fmt.Errorf("delete plan: %w", err)
	}
	return nil
}

func (r *PlanRepository) GetAll(ctx context.Context) ([]entities.Plan, error) {
	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return nil, err
	}
	rows, err := q.GetAllPlans(ctx)
	if err != nil {
		return nil, fmt.Errorf("get plans: %w", err)
	}
	result := make([]entities.Plan, 0, len(rows))
	for _, row := range rows {
		plan, err := mapPlan(row)
		if err != nil {
			return nil, err
		}
		result = append(result, plan)
	}
	return result, nil
}

func (r *PlanRepository) GetByID(ctx context.Context, id uuid.UUID) (entities.Plan, error) {
	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return entities.Plan{}, err
	}
	row, err := q.GetPlanByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if errors.Is(err, pgx.ErrNoRows) {
		return entities.Plan{}, application.ErrPlanNotFound
	}
	if err != nil {
		return entities.Plan{}, fmt.Errorf("get plan: %w", err)
	}
	return mapPlan(row)
}

func planPromotionParams(p *entities.Promotion) (
	pgtype.Text, pgtype.Int8, pgtype.Timestamptz, pgtype.Timestamptz,
) {
	if p == nil {
		return pgtype.Text{}, pgtype.Int8{}, pgtype.Timestamptz{}, pgtype.Timestamptz{}
	}
	return pgtype.Text{String: p.Name, Valid: true},
		pgtype.Int8{Int64: p.DiscountAmountMinor, Valid: true},
		pgtype.Timestamptz{Time: p.StartsAt, Valid: true},
		pgtype.Timestamptz{Time: p.EndsAt, Valid: true}
}

func mapPlan(row dbgen.Plan) (entities.Plan, error) {
	price, err := value_objects.NewMoney(row.PriceAmountMinor, row.Currency)
	if err != nil {
		return entities.Plan{}, fmt.Errorf("map plan price: %w", err)
	}
	plan := entities.Plan{
		ID: uuid.UUID(row.ID.Bytes), Type: row.Type, Name: row.Name,
		Description: row.Description, Price: price,
		Duration: time.Duration(row.DurationNs),
		Capabilities: entities.PlanCapabilities{
			MaxProducts: int(row.MaxProducts), AIEnabled: row.AiEnabled,
		},
		Available: row.Available,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
	if row.PromotionName.Valid {
		plan.Promotion = &entities.Promotion{
			Name:                row.PromotionName.String,
			DiscountAmountMinor: row.PromotionDiscountAmountMinor.Int64,
			StartsAt:            row.PromotionStartsAt.Time,
			EndsAt:              row.PromotionEndsAt.Time,
		}
	}
	return plan, nil
}

var _ ports.PlanRepository = (*PlanRepository)(nil)
