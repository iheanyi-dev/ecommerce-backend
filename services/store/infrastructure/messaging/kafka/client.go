package kafka

import (
	"context"
	"encoding/json"

	"go.opentelemetry.io/otel/propagation"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/shared/config"
)

// Producer is the subset of the franz-go client required by the Store
// Kafka producer adapter.
type Producer interface {
	ProduceSync(context.Context, ...*kgo.Record) kgo.ProduceResults
	Close()
}

// Client is the Store service Kafka producer.
type Client struct {
	client Producer
}

// NewClient creates the Kafka producer and manages its lifecycle.
func NewClient(lc fx.Lifecycle, cfg *config.Config) (*Client, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.KafkaBrokers),
	)
	if err != nil {
		return nil, err
	}

	c := &Client{
		client: client,
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			client.Close()
			return nil
		},
	})

	return c, nil
}

// NewClientWithProducer creates a Kafka Client around an injected producer.
//
// It is primarily intended for infrastructure unit tests where a live Kafka
// broker is not required.
func NewClientWithProducer(client Producer) *Client {
	return &Client{
		client: client,
	}
}

// Publish marshals the supplied event as JSON, injects the current W3C trace
// context into the Kafka record headers, and publishes the record.
func (c *Client) Publish(
	ctx context.Context,
	topic string,
	key string,
	value any,
) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}

	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(key),
		Value: payload,
	}

	injectTraceContext(ctx, record)

	return c.client.ProduceSync(ctx, record).FirstErr()
}

// injectTraceContext propagates the current W3C trace context into Kafka
// headers so downstream consumers can continue the distributed trace.
func injectTraceContext(ctx context.Context, record *kgo.Record) {
	carrier := propagation.MapCarrier{}

	propagation.TraceContext{}.Inject(ctx, carrier)

	for key, value := range carrier {
		record.Headers = append(record.Headers, kgo.RecordHeader{
			Key:   key,
			Value: []byte(value),
		})
	}
}
