package ports

import "context"

// SubscriptionOutboxEvent is a persisted event awaiting broker delivery.
type SubscriptionOutboxEvent struct {
	ID       string
	Payload  []byte
	Attempts int32
}

// SubscriptionOutboxRepository persists Store-facing subscription events.
type SubscriptionOutboxRepository interface {
	Create(ctx context.Context, eventID, subscriptionID, eventType string, payload []byte) error
	GetPending(ctx context.Context, limit int32) ([]SubscriptionOutboxEvent, error)
	MarkPublished(ctx context.Context, eventID string) error
	RecordFailure(ctx context.Context, eventID string, cause error) error
}
