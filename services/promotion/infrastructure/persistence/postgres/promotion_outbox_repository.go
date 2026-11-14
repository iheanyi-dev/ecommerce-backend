package postgres

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/ports"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/iheanyi-dev/ecommerce-backend/services/promotion/infrastructure/persistence/postgres/generated"
)

type PromotionOutboxRepository struct {
	pool *pgxpool.Pool
}

func NewPromotionOutboxRepository(
	pool *pgxpool.Pool,
) (*PromotionOutboxRepository, error) {
	if pool == nil {
		return nil, fmt.Errorf("promotion outbox repository requires a database pool")
	}

	return &PromotionOutboxRepository{pool: pool}, nil
}

func (r *PromotionOutboxRepository) Create(
	ctx context.Context,
	eventID string,
	promotionID string,
	eventType string,
	payload []byte,
) error {
	if eventType == "" {
		return fmt.Errorf("promotion event type is required")
	}
	if len(payload) == 0 {
		return fmt.Errorf("promotion event payload is required")
	}

	id, err := parseUUID(eventID)
	if err != nil {
		return err
	}

	promoID, err := parseUUID(promotionID)
	if err != nil {
		return err
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return err
	}

	if err := q.CreatePromotionOutbox(ctx, db.CreatePromotionOutboxParams{
		EventID:     id,
		PromotionID: promoID,
		EventType:   eventType,
		Payload:     append([]byte(nil), payload...),
	}); err != nil {
		return fmt.Errorf("create promotion outbox event: %w", err)
	}

	return nil
}

func (r *PromotionOutboxRepository) ClaimPending(
	ctx context.Context,
	limit int,
	lease time.Duration,
) ([]ports.PendingPromotionEvent, error) {
	if limit <= 0 || int64(limit) > int64(^uint32(0)>>1) {
		return nil, fmt.Errorf("invalid promotion outbox limit")
	}
	if lease <= 0 {
		return nil, fmt.Errorf("promotion outbox claim lease must be positive")
	}

	claimToken, err := newClaimToken()
	if err != nil {
		return nil, fmt.Errorf("generate promotion outbox claim token: %w", err)
	}

	now := time.Now().UTC()

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return nil, err
	}

	rows, err := q.ClaimPendingPromotionOutbox(
		ctx,
		db.ClaimPendingPromotionOutboxParams{
			ClaimToken: claimToken,
			LeaseUntil: pgtype.Timestamptz{
				Time:  now.Add(lease),
				Valid: true,
			},
			Now: pgtype.Timestamptz{
				Time:  now,
				Valid: true,
			},
			BatchSize: int32(limit),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("claim pending promotion events: %w", err)
	}

	result := make([]ports.PendingPromotionEvent, 0, len(rows))
	for _, row := range rows {
		result = append(result, ports.PendingPromotionEvent{
			EventID:     uuidString(row.EventID),
			PromotionID: uuidString(row.PromotionID),
			EventType:   row.EventType,
			Payload:     append([]byte(nil), row.Payload...),
			Attempts:    row.Attempts,
			ClaimToken:  uuidString(claimToken),
		})
	}

	return result, nil
}

func (r *PromotionOutboxRepository) MarkPublished(
	ctx context.Context,
	eventID string,
	claimToken string,
) error {
	id, err := parseUUID(eventID)
	if err != nil {
		return err
	}

	token, err := parseUUID(claimToken)
	if err != nil {
		return err
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return err
	}

	rows, err := q.MarkPromotionOutboxPublished(
		ctx,
		db.MarkPromotionOutboxPublishedParams{
			EventID:    id,
			ClaimToken: token,
		},
	)
	if err != nil {
		return fmt.Errorf("mark promotion event published: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("mark promotion event published: event not found or claim expired")
	}

	return nil
}

func (r *PromotionOutboxRepository) RecordFailure(
	ctx context.Context,
	eventID string,
	claimToken string,
	cause error,
) error {
	if cause == nil {
		return fmt.Errorf("promotion event failure cause is required")
	}

	id, err := parseUUID(eventID)
	if err != nil {
		return err
	}

	token, err := parseUUID(claimToken)
	if err != nil {
		return err
	}

	q, err := queriesFor(ctx, r.pool)
	if err != nil {
		return err
	}

	rows, err := q.RecordPromotionOutboxFailure(
		ctx,
		db.RecordPromotionOutboxFailureParams{
			EventID:    id,
			ClaimToken: token,
			LastError: pgtype.Text{
				String: cause.Error(),
				Valid:  true,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("record promotion event failure: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("record promotion event failure: event not found or claim expired")
	}

	return nil
}

func newClaimToken() (pgtype.UUID, error) {
	var token pgtype.UUID

	if _, err := rand.Read(token.Bytes[:]); err != nil {
		return pgtype.UUID{}, err
	}

	// Set RFC 4122 version 4 and variant bits.
	token.Bytes[6] = (token.Bytes[6] & 0x0f) | 0x40
	token.Bytes[8] = (token.Bytes[8] & 0x3f) | 0x80
	token.Valid = true

	return token, nil
}

var _ ports.PromotionOutboxRepository = (*PromotionOutboxRepository)(nil)
