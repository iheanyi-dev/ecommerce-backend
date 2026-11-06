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

	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/shared/config"
)

type paymentStatusChangedEvent struct {
	EventID               string `json:"event_id"`
	BusinessReferenceType string `json:"business_reference_type"`
	BusinessReference     string `json:"business_reference"`
	Status                string `json:"status"`
}

type PaymentConsumer struct {
	client *kgo.Client
	update *use_cases.UpdateSub
}

func NewPaymentConsumer(
	lifecycle fx.Lifecycle,
	cfg *config.Config,
	update *use_cases.UpdateSub,
) (*PaymentConsumer, error) {
	if cfg == nil || len(cfg.KafkaBrokers) == 0 ||
		cfg.PaymentEventsTopic == "" || cfg.PaymentConsumerGroup == "" {
		return nil, fmt.Errorf("Billing payment consumer configuration is incomplete")
	}
	client, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.KafkaBrokers...),
		kgo.ConsumerGroup(cfg.PaymentConsumerGroup),
		kgo.ConsumeTopics(cfg.PaymentEventsTopic),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		return nil, err
	}
	consumer := &PaymentConsumer{client: client, update: update}
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
			log.Printf("billing payment consumer: fetch error: %v", err)
		})
		fetches.EachRecord(func(record *kgo.Record) {
			for ctx.Err() == nil {
				if err := c.handleRecord(ctx, record); err != nil {
					log.Printf("billing payment consumer: processing failed; retrying: %v", err)
					select {
					case <-ctx.Done():
						return
					case <-time.After(2 * time.Second):
					}
					continue
				}

				// Do not process later records until this offset is committed.
				for ctx.Err() == nil {
					if err := c.client.CommitRecords(ctx, record); err == nil {
						return
					} else {
						log.Printf("billing payment consumer: commit failed; retrying: %v", err)
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

// HandlePaymentRecord processes one payment event without requiring a running Kafka consumer.
func HandlePaymentRecord(ctx context.Context, record *kgo.Record, update *use_cases.UpdateSub) error {
	return (&PaymentConsumer{update: update}).handleRecord(ctx, record)
}

func (c *PaymentConsumer) handleRecord(ctx context.Context, record *kgo.Record) error {
	if record == nil {
		return fmt.Errorf("payment event record is nil")
	}

	var event paymentStatusChangedEvent
	if err := json.Unmarshal(record.Value, &event); err != nil {
		log.Printf("billing payment consumer: discarding malformed event: %v", err)
		return nil
	}
	if event.BusinessReferenceType != ports.BusinessReferenceTypeSubscription {
		return nil
	}

	var status entities.SubscriptionStatus
	switch event.Status {
	case "succeeded":
		status = entities.SubscriptionStatusActive
	case "failed":
		status = entities.SubscriptionStatusFailed
	default:
		log.Printf("billing payment consumer: ignoring non-terminal status %q", event.Status)
		return nil
	}
	if event.BusinessReference == "" {
		log.Printf("billing payment consumer: discarding event without business reference")
		return nil
	}
	subscriptionID, err := uuid.Parse(event.BusinessReference)
	if err != nil || subscriptionID == uuid.Nil {
		log.Printf("billing payment consumer: discarding event with invalid subscription reference")
		return nil
	}
	if c.update == nil {
		return fmt.Errorf("billing subscription update use case is unavailable")
	}

	_, err = c.update.Execute(ctx, use_cases.UpdateSubInput{
		BusinessReferenceID: event.BusinessReference,
		Status:              status,
	})
	return err
}
