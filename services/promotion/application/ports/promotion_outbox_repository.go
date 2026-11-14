package ports

import (
	"context"
	"time"
)

// PendingPromotionEvent is an event claimed for delivery to Kafka.
type PendingPromotionEvent struct {
	EventID     string
	PromotionID string
	EventType   string
	Payload     []byte
	Attempts    int32
	ClaimToken  string
}

// PromotionOutboxRepository persists events atomically with promotion changes.
type PromotionOutboxRepository interface {
	Create(
		ctx context.Context,
		eventID string,
		promotionID string,
		eventType string,
		payload []byte,
	) error

	ClaimPending(ctx context.Context, limit int, lease time.Duration) ([]PendingPromotionEvent, error)
	MarkPublished(ctx context.Context, eventID, claimToken string) error
	RecordFailure(ctx context.Context, eventID, claimToken string, cause error) error
}
