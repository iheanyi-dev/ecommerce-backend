package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/shared/config"
)

// Client publishes Kafka records and closes its producer with the Fx lifecycle.
type Client struct {
	producer Producer
}

func NewClient(lc fx.Lifecycle, cfg *config.Config) (*Client, error) {
	producer, err := kgo.NewClient(kgo.SeedBrokers(cfg.KafkaBrokers))
	if err != nil {
		return nil, fmt.Errorf("create product Kafka producer: %w", err)
	}
	client := &Client{producer: producer}
	lc.Append(fx.Hook{OnStop: func(context.Context) error {
		producer.Close()
		return nil
	}})
	return client, nil
}

// NewClientWithProducer supports focused unit tests.
func NewClientWithProducer(producer Producer) *Client {
	return &Client{producer: producer}
}

func (c *Client) Publish(ctx context.Context, topic, key string, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal Kafka event for topic %q: %w", topic, err)
	}
	record := &kgo.Record{Topic: topic, Key: []byte(key), Value: payload}
	carrier := propagation.MapCarrier{}
	propagation.TraceContext{}.Inject(ctx, carrier)
	for name, value := range carrier {
		record.Headers = append(record.Headers, kgo.RecordHeader{Key: name, Value: []byte(value)})
	}
	if err := c.producer.ProduceSync(ctx, record).FirstErr(); err != nil {
		return fmt.Errorf("publish Kafka event to topic %q: %w", topic, err)
	}
	return nil
}
