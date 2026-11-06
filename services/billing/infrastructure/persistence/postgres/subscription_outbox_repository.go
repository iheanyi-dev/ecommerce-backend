package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	dbgen "github.com/iheanyi-dev/ecommerce-backend/services/billing/infrastructure/persistence/postgres/generated"
)

type SubscriptionOutboxRepository struct {
	pool *pgxpool.Pool
}

func NewSubscriptionOutboxRepository(pool *pgxpool.Pool) (*SubscriptionOutboxRepository, error) {
	if pool == nil {
		return nil, fmt.Errorf("billing database pool is required")
	}
	return &SubscriptionOutboxRepository{pool: pool}, nil
}

func (r *SubscriptionOutboxRepository) Create(
	ctx context.Context, eventID, subscriptionID, eventType string, payload []byte,
) error {
	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		return fmt.Errorf("parse event ID: %w", err)
	}
	subscriptionUUID, err := uuid.Parse(subscriptionID)
	if err != nil {
		return fmt.Errorf("parse subscription ID: %w", err)
	}
	if eventType == "" || len(payload) == 0 {
		return fmt.Errorf("event type and payload are required")
	}
	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return err
	}
	err = q.CreateSubscriptionOutboxEvent(ctx, dbgen.CreateSubscriptionOutboxEventParams{
		EventID:        pgtype.UUID{Bytes: eventUUID, Valid: true},
		SubscriptionID: pgtype.UUID{Bytes: subscriptionUUID, Valid: true},
		EventType:      eventType, Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("create subscription outbox event: %w", err)
	}
	return nil
}

func (r *SubscriptionOutboxRepository) GetPending(
	ctx context.Context, limit int32,
) ([]ports.SubscriptionOutboxEvent, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("pending event limit must be positive")
	}
	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return nil, err
	}
	rows, err := q.GetPendingSubscriptionOutboxEvents(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("get pending subscription events: %w", err)
	}
	result := make([]ports.SubscriptionOutboxEvent, 0, len(rows))
	for _, row := range rows {
		if !row.EventID.Valid {
			return nil, fmt.Errorf("pending subscription event has invalid ID")
		}
		result = append(result, ports.SubscriptionOutboxEvent{
			ID:       uuid.UUID(row.EventID.Bytes).String(),
			Payload:  append([]byte(nil), row.Payload...),
			Attempts: row.Attempts,
		})
	}
	return result, nil
}

func (r *SubscriptionOutboxRepository) MarkPublished(ctx context.Context, eventID string) error {
	id, err := uuid.Parse(eventID)
	if err != nil {
		return fmt.Errorf("parse outbox event ID: %w", err)
	}
	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return err
	}
	if err := q.MarkSubscriptionOutboxEventPublished(ctx, pgtype.UUID{Bytes: id, Valid: true}); err != nil {
		return fmt.Errorf("mark subscription event published: %w", err)
	}
	return nil
}

func (r *SubscriptionOutboxRepository) RecordFailure(
	ctx context.Context, eventID string, cause error,
) error {
	id, err := uuid.Parse(eventID)
	if err != nil {
		return fmt.Errorf("parse outbox event ID: %w", err)
	}
	if cause == nil {
		return fmt.Errorf("delivery failure cause is required")
	}
	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return err
	}
	if err := q.RecordSubscriptionOutboxEventFailure(ctx, dbgen.RecordSubscriptionOutboxEventFailureParams{
		EventID:   pgtype.UUID{Bytes: id, Valid: true},
		LastError: pgtype.Text{String: cause.Error(), Valid: true},
	}); err != nil {
		return fmt.Errorf("record subscription event failure: %w", err)
	}
	return nil
}

var _ ports.SubscriptionOutboxRepository = (*SubscriptionOutboxRepository)(nil)
