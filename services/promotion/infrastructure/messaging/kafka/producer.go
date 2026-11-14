package kafka

import (
	"context"
	"fmt"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/ports"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/fx"
)

type ProducerClient interface {
	ProduceSync(ctx context.Context, record *kgo.Record) error
	Close()
}

type franzProducerClient struct {
	client *kgo.Client
}

func (c *franzProducerClient) ProduceSync(
	ctx context.Context,
	record *kgo.Record,
) error {
	return c.client.ProduceSync(ctx, record).FirstErr()
}

func (c *franzProducerClient) Close() {
	c.client.Close()
}

type Producer struct {
	client ProducerClient
	topic  string
}

func NewProducer(
	lifecycle fx.Lifecycle,
	brokers []string,
	topic string,
) (*Producer, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("promotion Kafka brokers are required")
	}
	if topic == "" {
		return nil, fmt.Errorf("promotion Kafka topic is required")
	}

	client, err := kgo.NewClient(kgo.SeedBrokers(brokers...))
	if err != nil {
		return nil, fmt.Errorf("create promotion Kafka producer: %w", err)
	}

	producer, err := NewProducerWithClient(&franzProducerClient{client: client}, topic)
	if err != nil {
		client.Close()
		return nil, err
	}

	lifecycle.Append(fx.Hook{
		OnStop: func(context.Context) error {
			producer.client.Close()
			return nil
		},
	})

	return producer, nil
}

func NewProducerWithClient(
	client ProducerClient,
	topic string,
) (*Producer, error) {
	if client == nil {
		return nil, fmt.Errorf("promotion Kafka producer client is required")
	}
	if topic == "" {
		return nil, fmt.Errorf("promotion Kafka topic is required")
	}

	return &Producer{
		client: client,
		topic:  topic,
	}, nil
}

func (p *Producer) Publish(
	ctx context.Context,
	event ports.PendingPromotionEvent,
) error {
	if event.EventID == "" {
		return fmt.Errorf("promotion event ID is required")
	}
	if event.PromotionID == "" {
		return fmt.Errorf("promotion ID is required")
	}
	if event.EventType == "" {
		return fmt.Errorf("promotion event type is required")
	}
	if len(event.Payload) == 0 {
		return fmt.Errorf("promotion event payload is required")
	}

	record := &kgo.Record{
		Topic: p.topic,
		Key:   []byte(event.PromotionID),
		Value: append([]byte(nil), event.Payload...),
		Headers: []kgo.RecordHeader{
			{
				Key:   "event_type",
				Value: []byte(event.EventType),
			},
		},
	}

	if err := p.client.ProduceSync(ctx, record); err != nil {
		return fmt.Errorf("publish promotion event %s: %w", event.EventID, err)
	}

	return nil
}

var _ ports.PromotionEventPublisher = (*Producer)(nil)
