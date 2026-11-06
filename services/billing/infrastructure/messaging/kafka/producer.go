package kafka

import (
	"context"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/shared/config"
)

type Producer struct {
	client *kgo.Client
	topic  string
}

func NewProducer(lifecycle fx.Lifecycle, cfg *config.Config) (*Producer, error) {
	if cfg == nil || len(cfg.KafkaBrokers) == 0 || cfg.StoreEventsTopic == "" {
		return nil, fmt.Errorf("Kafka brokers and Store events topic are required")
	}
	client, err := kgo.NewClient(kgo.SeedBrokers(cfg.KafkaBrokers...))
	if err != nil {
		return nil, fmt.Errorf("create Billing Kafka producer: %w", err)
	}
	producer := &Producer{client: client, topic: cfg.StoreEventsTopic}
	lifecycle.Append(fx.Hook{OnStop: func(context.Context) error {
		producer.Close()
		return nil
	}})
	return producer, nil
}

func (p *Producer) Publish(ctx context.Context, event ports.SubscriptionOutboxEvent) error {
	if event.ID == "" || len(event.Payload) == 0 {
		return fmt.Errorf("outbox event ID and payload are required")
	}
	err := p.client.ProduceSync(ctx, &kgo.Record{
		Topic: p.topic, Key: []byte(event.ID), Value: event.Payload,
	}).FirstErr()
	if err != nil {
		return fmt.Errorf("publish subscription event %s: %w", event.ID, err)
	}
	return nil
}

func (p *Producer) Close() { p.client.Close() }
