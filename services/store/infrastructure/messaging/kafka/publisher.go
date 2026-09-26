package kafka

import "context"

// MessagePublisher publishes messages to a Kafka-compatible broker.
type MessagePublisher interface {
	Publish(ctx context.Context, topic string, key string, value any) error
}
