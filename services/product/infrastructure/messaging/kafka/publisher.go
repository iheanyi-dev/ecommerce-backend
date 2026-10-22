package kafka

import (
	"context"

	"github.com/twmb/franz-go/pkg/kgo"
)

// MessagePublisher publishes keyed JSON-compatible messages to Kafka.
type MessagePublisher interface {
	Publish(ctx context.Context, topic, key string, value any) error
}

type Producer interface {
	ProduceSync(context.Context, ...*kgo.Record) kgo.ProduceResults
	Close()
}
