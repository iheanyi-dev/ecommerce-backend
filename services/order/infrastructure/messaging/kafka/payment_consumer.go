package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/shared/config"
)

const (
	paymentBusinessReferenceTypeOrder = "order"
	paymentStatusChangedEventType     = "payment.status_changed"
	payoutStatusChangedEventType      = "payout.status_changed"
)

type paymentStatusChangedEvent struct {
	EventID               string `json:"event_id"`
	BusinessReferenceType string `json:"business_reference_type"`
	BusinessReference     string `json:"business_reference"`
	Status                string `json:"status"`
}

type payoutStatusChangedEvent struct {
	EventID  string `json:"event_id"`
	PayoutID string `json:"payout_id"`
	OrderID  string `json:"order_id"`
	Status   string `json:"status"`
}

type PaymentConsumer struct {
	client  *kgo.Client
	handler ports.PaymentEventHandler
}

func NewPaymentConsumer(
	lifecycle fx.Lifecycle,
	cfg *config.Config,
	handler ports.PaymentEventHandler,
) (*PaymentConsumer, error) {
	if cfg == nil || len(cfg.KafkaBrokers) == 0 ||
		cfg.PaymentEventsTopic == "" || cfg.PaymentConsumerGroup == "" {
		return nil, fmt.Errorf("order payment consumer configuration is incomplete")
	}
	if handler == nil {
		return nil, fmt.Errorf("order payment event handler is nil")
	}

	client, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.KafkaBrokers...),
		kgo.ConsumerGroup(cfg.PaymentConsumerGroup),
		kgo.ConsumeTopics(cfg.PaymentEventsTopic),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		return nil, fmt.Errorf("create order payment Kafka consumer: %w", err)
	}

	consumer := &PaymentConsumer{client: client, handler: handler}
	ctx, cancel := context.WithCancel(context.Background())
	var done chan struct{}

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			done = make(chan struct{})
			go func() {
				defer close(done)
				consumer.run(ctx)
			}()
			return nil
		},
		OnStop: func(stopCtx context.Context) error {
			cancel()
			client.Close()
			select {
			case <-done:
				return nil
			case <-stopCtx.Done():
				return stopCtx.Err()
			}
		},
	})

	return consumer, nil
}

func (c *PaymentConsumer) run(ctx context.Context) {
	for {
		fetches := c.client.PollFetches(ctx)
		if fetches.IsClientClosed() || ctx.Err() != nil {
			return
		}

		fetches.EachError(func(_ string, _ int32, err error) {
			log.Printf("order payment consumer: fetch error: %v", err)
		})

		fetches.EachRecord(func(record *kgo.Record) {
			for ctx.Err() == nil {
				if err := c.handleRecord(ctx, record); err != nil {
					log.Printf("order payment consumer: processing failed; retrying: %v", err)
					select {
					case <-ctx.Done():
						return
					case <-time.After(2 * time.Second):
					}
					continue
				}

				for ctx.Err() == nil {
					if err := c.client.CommitRecords(ctx, record); err == nil {
						return
					} else {
						log.Printf("order payment consumer: commit failed; retrying: %v", err)
					}

					select {
					case <-ctx.Done():
						return
					case <-time.After(2 * time.Second):
					}
				}
			}
		})
	}
}

// HandlePaymentRecord processes one event without starting a Kafka consumer.
func HandlePaymentRecord(
	ctx context.Context,
	record *kgo.Record,
	handler ports.PaymentEventHandler,
) error {
	return (&PaymentConsumer{handler: handler}).handleRecord(ctx, record)
}

func (c *PaymentConsumer) handleRecord(ctx context.Context, record *kgo.Record) error {
	if record == nil {
		return fmt.Errorf("payment event record is nil")
	}
	if c.handler == nil {
		return fmt.Errorf("order payment event handler is unavailable")
	}

	eventType := ""
	for _, header := range record.Headers {
		if header.Key == "event_type" {
			eventType = string(header.Value)
			break
		}
	}

	switch eventType {
	case paymentStatusChangedEventType:
		return c.handlePaymentEvent(ctx, record.Value)
	case payoutStatusChangedEventType:
		return c.handlePayoutEvent(ctx, record.Value)
	default:
		log.Printf("order payment consumer: discarding event with missing or unsupported event type %q", eventType)
		return nil
	}
}

func (c *PaymentConsumer) handlePaymentEvent(ctx context.Context, payload []byte) error {
	var event paymentStatusChangedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		log.Printf("order payment consumer: discarding malformed payment event: %v", err)
		return nil
	}
	if event.BusinessReferenceType != paymentBusinessReferenceTypeOrder {
		return nil
	}
	if event.Status != "succeeded" && event.Status != "failed" {
		return nil
	}
	if !validUUID(event.BusinessReference) {
		log.Printf("order payment consumer: discarding payment event with invalid order reference")
		return nil
	}

	return c.handler.HandlePayment(ctx, event.BusinessReference, event.Status)
}

func (c *PaymentConsumer) handlePayoutEvent(ctx context.Context, payload []byte) error {
	var event payoutStatusChangedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		log.Printf("order payment consumer: discarding malformed payout event: %v", err)
		return nil
	}
	if event.Status != "succeeded" && event.Status != "failed" {
		return nil
	}
	if !validUUID(event.OrderID) {
		log.Printf("order payment consumer: discarding payout event with invalid order ID")
		return nil
	}

	return c.handler.HandlePayout(ctx, event.OrderID, event.Status)
}

func validUUID(value string) bool {
	id, err := uuid.Parse(value)
	return err == nil && id != uuid.Nil
}
