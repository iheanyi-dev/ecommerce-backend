package kafka

import (
	"context"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/shared/config"
)

// Producer publishes payment events to Kafka.
type ProducerClient interface {
	ProduceSync(context.Context, ...*kgo.Record) kgo.ProduceResults
	Close()
}

// Producer publishes payment events to Kafka.
type Producer struct {
	client ProducerClient
	topic  string
}

// NewProducer creates the Kafka producer using centrally loaded service configuration.
func NewProducer(lifecycle fx.Lifecycle, cfg *config.Config) (*Producer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Payment configuration is required")
	}
	if len(cfg.KafkaBrokers) == 0 {
		return nil, fmt.Errorf("at least one Kafka broker is required")
	}
	if cfg.PaymentEventsTopic == "" {
		return nil, fmt.Errorf("payment events topic is required")
	}

	client, err := kgo.NewClient(kgo.SeedBrokers(cfg.KafkaBrokers...))
	if err != nil {
		return nil, fmt.Errorf("create Kafka producer: %w", err)
	}

	producer := &Producer{
		client: client,
		topic:  cfg.PaymentEventsTopic,
	}

	lifecycle.Append(fx.Hook{
		OnStop: func(context.Context) error {
			producer.Close()
			return nil
		},
	})

	return producer, nil
}

// NewProducerWithClient creates a producer using an injected Kafka client.
func NewProducerWithClient(client ProducerClient, topic string) (*Producer, error) {
	if client == nil {
		return nil, fmt.Errorf("Kafka producer client is required")
	}
	if topic == "" {
		return nil, fmt.Errorf("payment events topic is required")
	}

	return &Producer{client: client, topic: topic}, nil
}

// Publish sends an outbox event to the configured Kafka topic.
func (p *Producer) Publish(ctx context.Context, event ports.PaymentOutboxEvent) error {
	if event.ID == "" || event.EventType == "" || len(event.Payload) == 0 {
		return fmt.Errorf("outbox event ID, event type, and payload are required")
	}

	record := &kgo.Record{
		Topic: p.topic,
		Key:   []byte(event.ID),
		Value: event.Payload,
		Headers: []kgo.RecordHeader{
			{Key: "event_type", Value: []byte(event.EventType)},
		},
	}

	if err := p.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		return fmt.Errorf("publish payment event %s: %w", event.ID, err)
	}

	return nil
}

// Close releases the Kafka producer resources.
func (p *Producer) Close() {
	p.client.Close()
}
