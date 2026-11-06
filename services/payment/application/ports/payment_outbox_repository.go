package ports

import "context"

// PaymentOutboxEvent represents a payment event waiting for Kafka delivery.
type PaymentOutboxEvent struct {
	ID        string
	EventType string
	Payload   []byte
	Attempts  int32
}

// PaymentOutboxRepository persists payment events for reliable delivery.
type PaymentOutboxRepository interface {
	Create(ctx context.Context, eventID, aggregateType, aggregateID, deduplicationKey, eventType string, payload []byte) error
	GetPending(ctx context.Context, limit int32) ([]PaymentOutboxEvent, error)
	MarkPublished(ctx context.Context, eventID string) error
	RecordFailure(ctx context.Context, eventID string, cause error) error
}
