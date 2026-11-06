package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/value_objects"
	dbgen "github.com/iheanyi-dev/ecommerce-backend/services/billing/infrastructure/persistence/postgres/generated"
)

// SubscriptionRepository persists Billing subscriptions.
type SubscriptionRepository struct {
	pool *pgxpool.Pool
}

// NewSubscriptionRepository creates a PostgreSQL subscription repository.
func NewSubscriptionRepository(pool *pgxpool.Pool) (*SubscriptionRepository, error) {
	if pool == nil {
		return nil, fmt.Errorf("billing subscription repository requires a database pool")
	}
	return &SubscriptionRepository{pool: pool}, nil
}

// GetByID retrieves a subscription by its ID.
func (r *SubscriptionRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*entities.Subscription, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("subscription ID is required")
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return nil, err
	}

	row, err := q.GetSubscriptionByIDForUpdate(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, application.ErrSubscriptionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get subscription by ID: %w", err)
	}

	subscription, err := mapSubscription(row)
	if err != nil {
		return nil, fmt.Errorf("map subscription: %w", err)
	}
	return subscription, nil
}

// GetCurrentByStore retrieves and locks the store's pending or active subscription.
func (r *SubscriptionRepository) GetCurrentByStore(
	ctx context.Context,
	storeID uuid.UUID,
) (*entities.Subscription, error) {
	if storeID == uuid.Nil {
		return nil, fmt.Errorf("store ID is required")
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return nil, err
	}

	row, err := q.GetCurrentSubscriptionByStore(
		ctx,
		pgtype.UUID{Bytes: storeID, Valid: true},
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, application.ErrSubscriptionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get current subscription by store: %w", err)
	}

	subscription, err := mapSubscription(row)
	if err != nil {
		return nil, fmt.Errorf("map current subscription: %w", err)
	}
	return subscription, nil
}

// GetDueForUpdate retrieves and locks a batch of expired active subscriptions.
func (r *SubscriptionRepository) GetDueForUpdate(
	ctx context.Context,
	now time.Time,
	limit int32,
) ([]entities.Subscription, error) {
	if now.IsZero() || limit <= 0 {
		return nil, fmt.Errorf("valid expiration time and positive limit are required")
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return nil, err
	}

	rows, err := q.GetDueSubscriptionsForUpdate(ctx, dbgen.GetDueSubscriptionsForUpdateParams{
		ExpiresAt: pgtype.Timestamptz{Time: now.UTC(), Valid: true},
		Limit:     limit,
	})
	if err != nil {
		return nil, fmt.Errorf("get due subscriptions: %w", err)
	}

	result := make([]entities.Subscription, 0, len(rows))
	for _, row := range rows {
		subscription, err := mapSubscription(row)
		if err != nil {
			return nil, fmt.Errorf("map due subscription: %w", err)
		}
		result = append(result, *subscription)
	}
	return result, nil
}

// GetByStoreID retrieves all subscriptions belonging to a store.
func (r *SubscriptionRepository) GetByStoreID(
	ctx context.Context,
	storeID uuid.UUID,
) ([]entities.Subscription, error) {
	if storeID == uuid.Nil {
		return nil, fmt.Errorf("store ID is required")
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return nil, err
	}

	rows, err := q.GetSubscriptionsByStoreID(
		ctx,
		pgtype.UUID{Bytes: storeID, Valid: true},
	)
	if err != nil {
		return nil, fmt.Errorf("get subscriptions by store ID: %w", err)
	}

	subscriptions := make([]entities.Subscription, 0, len(rows))
	for _, row := range rows {
		subscription, err := mapSubscription(row)
		if err != nil {
			return nil, fmt.Errorf("map subscription in store list: %w", err)
		}
		subscriptions = append(subscriptions, *subscription)
	}

	return subscriptions, nil
}

// Create persists a new subscription.
func (r *SubscriptionRepository) Create(
	ctx context.Context,
	subscription entities.Subscription,
) error {
	if err := validateSubscription(subscription); err != nil {
		return err
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return err
	}

	err = q.CreateSubscription(ctx, dbgen.CreateSubscriptionParams{
		ID:               pgtype.UUID{Bytes: subscription.ID, Valid: true},
		StoreID:          pgtype.UUID{Bytes: subscription.StoreID, Valid: true},
		PlanID:           pgtype.UUID{Bytes: subscription.PlanID, Valid: true},
		PriceAmountMinor: subscription.Price.AmountMinor,
		Currency:         subscription.Price.Currency,
		DurationNs:       int64(subscription.Duration),
		Status:           string(subscription.Status),
		StartedAt:        nullableTimestamp(subscription.StartedAt),
		ExpiresAt:        nullableTimestamp(subscription.ExpiresAt),
		CreatedAt:        pgtype.Timestamptz{Time: subscription.CreatedAt, Valid: true},
		UpdatedAt:        pgtype.Timestamptz{Time: subscription.UpdatedAt, Valid: true},
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) &&
			pgErr.Code == "23505" &&
			pgErr.ConstraintName == "subscriptions_one_current_per_store_idx" {
			return application.ErrSubscriptionConflict
		}
		return fmt.Errorf("create subscription: %w", err)
	}

	return nil
}

// Update persists mutable subscription state.
func (r *SubscriptionRepository) Update(
	ctx context.Context,
	subscription entities.Subscription,
) error {
	if err := validateSubscription(subscription); err != nil {
		return err
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return err
	}

	err = q.UpdateSubscription(ctx, dbgen.UpdateSubscriptionParams{
		ID:        pgtype.UUID{Bytes: subscription.ID, Valid: true},
		Status:    string(subscription.Status),
		StartedAt: nullableTimestamp(subscription.StartedAt),
		ExpiresAt: nullableTimestamp(subscription.ExpiresAt),
		UpdatedAt: pgtype.Timestamptz{Time: subscription.UpdatedAt, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}

	return nil
}

func validateSubscription(subscription entities.Subscription) error {
	if subscription.ID == uuid.Nil ||
		subscription.StoreID == uuid.Nil ||
		subscription.PlanID == uuid.Nil ||
		subscription.Duration <= 0 ||
		subscription.CreatedAt.IsZero() ||
		subscription.UpdatedAt.IsZero() {
		return fmt.Errorf("invalid subscription persistence data")
	}

	money, err := value_objects.NewMoney(
		subscription.Price.AmountMinor,
		subscription.Price.Currency,
	)
	if err != nil || money != subscription.Price {
		return fmt.Errorf("invalid subscription price")
	}

	return nil
}

func nullableTimestamp(value time.Time) pgtype.Timestamptz {
	if value.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func mapSubscription(row dbgen.Subscription) (*entities.Subscription, error) {
	price, err := value_objects.NewMoney(row.PriceAmountMinor, row.Currency)
	if err != nil {
		return nil, fmt.Errorf("reconstitute subscription price: %w", err)
	}

	subscription := &entities.Subscription{
		ID:        uuid.UUID(row.ID.Bytes),
		StoreID:   uuid.UUID(row.StoreID.Bytes),
		PlanID:    uuid.UUID(row.PlanID.Bytes),
		Price:     price,
		Duration:  time.Duration(row.DurationNs),
		Status:    entities.SubscriptionStatus(row.Status),
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}

	if row.StartedAt.Valid {
		subscription.StartedAt = row.StartedAt.Time
	}
	if row.ExpiresAt.Valid {
		subscription.ExpiresAt = row.ExpiresAt.Time
	}

	return subscription, nil
}

var _ ports.SubscriptionRepository = (*SubscriptionRepository)(nil)
