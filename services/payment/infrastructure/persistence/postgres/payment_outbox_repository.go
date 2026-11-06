package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	dbgen "github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure/persistence/postgres/generated"
)

// PaymentOutboxRepository persists payment events in PostgreSQL.
type PaymentOutboxRepository struct {
	pool *pgxpool.Pool
}

// NewPaymentOutboxRepository creates a PostgreSQL payment outbox repository.
func NewPaymentOutboxRepository(pool *pgxpool.Pool) (*PaymentOutboxRepository, error) {
	if pool == nil {
		return nil, fmt.Errorf("payment database pool is required")
	}
	return &PaymentOutboxRepository{pool: pool}, nil
}

// Create stores an event in the same transaction as the payment when one exists.
func (r *PaymentOutboxRepository) Create(
	ctx context.Context,
	eventID, aggregateType, aggregateID, deduplicationKey, eventType string,
	payload []byte,
) error {
	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		return fmt.Errorf("parse event ID: %w", err)
	}
	aggregateUUID, err := uuid.Parse(aggregateID)
	if err != nil {
		return fmt.Errorf("parse aggregate ID: %w", err)
	}
	if aggregateType == "" || deduplicationKey == "" || eventType == "" || len(payload) == 0 {
		return fmt.Errorf("event type and payload are required")
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return fmt.Errorf("get payment queries: %w", err)
	}
	if err := q.CreatePaymentOutboxEvent(ctx, dbgen.CreatePaymentOutboxEventParams{
		EventID:          pgtype.UUID{Bytes: eventUUID, Valid: true},
		AggregateType:    aggregateType,
		AggregateID:      pgtype.UUID{Bytes: aggregateUUID, Valid: true},
		DeduplicationKey: deduplicationKey,
		EventType:        eventType,
		Payload:          payload,
	}); err != nil {
		return fmt.Errorf("create payment outbox event: %w", err)
	}
	return nil
}

// GetPending returns unpublished events in creation order.
func (r *PaymentOutboxRepository) GetPending(
	ctx context.Context,
	limit int32,
) ([]ports.PaymentOutboxEvent, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("pending event limit must be positive")
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return nil, fmt.Errorf("get payment queries: %w", err)
	}
	rows, err := q.GetPendingPaymentOutboxEvents(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("get pending payment outbox events: %w", err)
	}

	events := make([]ports.PaymentOutboxEvent, 0, len(rows))
	for _, row := range rows {
		if !row.EventID.Valid {
			return nil, fmt.Errorf("pending payment outbox event has an invalid ID")
		}
		events = append(events, ports.PaymentOutboxEvent{
			ID:        uuid.UUID(row.EventID.Bytes).String(),
			EventType: row.EventType,
			Payload:   append([]byte(nil), row.Payload...),
			Attempts:  row.Attempts,
		})
	}
	return events, nil
}

// MarkPublished records successful delivery.
func (r *PaymentOutboxRepository) MarkPublished(ctx context.Context, eventID string) error {
	id, err := uuid.Parse(eventID)
	if err != nil {
		return fmt.Errorf("parse outbox event ID: %w", err)
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return fmt.Errorf("get payment queries: %w", err)
	}
	if err := q.MarkPaymentOutboxEventPublished(ctx, pgtype.UUID{
		Bytes: id,
		Valid: true,
	}); err != nil {
		return fmt.Errorf("mark payment outbox event published: %w", err)
	}
	return nil
}

// RecordFailure records a failed delivery attempt.
func (r *PaymentOutboxRepository) RecordFailure(
	ctx context.Context,
	eventID string,
	cause error,
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
		return fmt.Errorf("get payment queries: %w", err)
	}
	if err := q.RecordPaymentOutboxEventFailure(ctx, dbgen.RecordPaymentOutboxEventFailureParams{
		EventID:   pgtype.UUID{Bytes: id, Valid: true},
		LastError: pgtype.Text{String: cause.Error(), Valid: true},
	}); err != nil {
		return fmt.Errorf("record payment outbox delivery failure: %w", err)
	}
	return nil
}

var _ ports.PaymentOutboxRepository = (*PaymentOutboxRepository)(nil)
