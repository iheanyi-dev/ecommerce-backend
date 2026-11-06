package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/shared/config"
)

const subscriptionUpdatedEventType = "billing.subscription.updated"

// subscriptionUpdatedEvent mirrors Billing's published event contract.
// Store intentionally owns this transport representation and does not import
// Billing's application or domain packages.
type subscriptionUpdatedEvent struct {
	EventID        string `json:"event_id"`
	SubscriptionID string `json:"subscription_id"`
	StoreID        string `json:"store_id"`
	PlanType       string `json:"plan_type"`
	Status         string `json:"status"`
}

// SubscriptionEventConsumer consumes trusted Billing subscription events.
type SubscriptionEventConsumer struct {
	client  *kgo.Client
	service ports.UpdateAfterSubscriptionService
	topic   string

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	once   sync.Once
}

// NewSubscriptionEventConsumer creates and registers the Billing event
// consumer. Automatic offset commits are disabled; records are committed
// only after the Store application use case succeeds.
func NewSubscriptionEventConsumer(
	lifecycle fx.Lifecycle,
	cfg *config.Config,
	service ports.UpdateAfterSubscriptionService,
) (*SubscriptionEventConsumer, error) {
	if cfg == nil {
		return nil, errors.New("subscription event consumer: config is required")
	}
	if cfg.KafkaBrokers == "" {
		return nil, errors.New("subscription event consumer: Kafka brokers are required")
	}
	if cfg.BillingStoreEventsTopic == "" {
		return nil, errors.New("subscription event consumer: topic is required")
	}
	if cfg.StoreSubscriptionConsumerGroup == "" {
		return nil, errors.New("subscription event consumer: consumer group is required")
	}
	if service == nil {
		return nil, errors.New("subscription event consumer: update service is required")
	}

	client, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.KafkaBrokers),
		kgo.ConsumerGroup(cfg.StoreSubscriptionConsumerGroup),
		kgo.ConsumeTopics(cfg.BillingStoreEventsTopic),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		return nil, fmt.Errorf("create subscription event Kafka client: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	consumer := &SubscriptionEventConsumer{
		client:  client,
		service: service,
		topic:   cfg.BillingStoreEventsTopic,
		ctx:     ctx,
		cancel:  cancel,
		done:    make(chan struct{}),
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go consumer.run()
			return nil
		},
		OnStop: func(context.Context) error {
			consumer.stop()
			return nil
		},
	})

	return consumer, nil
}

// run polls and processes one Kafka record at a time.
//
// A failed record is retried before another record is fetched. This prevents
// later records from advancing the committed offset beyond an unprocessed
// subscription event.
func (c *SubscriptionEventConsumer) run() {
	defer close(c.done)

	for {
		fetches := c.client.PollRecords(c.ctx, 1)
		if c.ctx.Err() != nil || fetches.IsClientClosed() {
			return
		}

		for _, fetchErr := range fetches.Errors() {
			slog.Error(
				"Store subscription event Kafka fetch failed",
				"topic", fetchErr.Topic,
				"partition", fetchErr.Partition,
				"error", fetchErr.Err,
			)
		}

		fetches.EachRecord(func(record *kgo.Record) {
			// Do not fetch another record until this event is applied.
			for c.ctx.Err() == nil {
				if err := c.handleRecord(c.ctx, record); err != nil {
					slog.Error(
						"Store subscription event processing failed; retrying",
						"topic", record.Topic,
						"partition", record.Partition,
						"offset", record.Offset,
						"error", err,
					)

					if !waitForSubscriptionRetry(c.ctx) {
						return
					}
					continue
				}

				break
			}

			if c.ctx.Err() != nil {
				return
			}

			// The application operation succeeded. Retry only the commit if
			// Kafka rejects it; do not execute the application operation again.
			for c.ctx.Err() == nil {
				if err := c.client.CommitRecords(c.ctx, record); err != nil {
					slog.Error(
						"Store subscription event offset commit failed; retrying",
						"topic", record.Topic,
						"partition", record.Partition,
						"offset", record.Offset,
						"error", err,
					)

					if !waitForSubscriptionRetry(c.ctx) {
						return
					}
					continue
				}

				break
			}
		})
	}
}

// waitForSubscriptionRetry pauses before retrying a failed event or offset
// commit. It returns false when consumer shutdown is requested.
func waitForSubscriptionRetry(ctx context.Context) bool {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// handleRecord validates the Billing event and delegates the state transition
// to the application layer.
func (c *SubscriptionEventConsumer) handleRecord(
	ctx context.Context,
	record *kgo.Record,
) error {
	if record == nil {
		return errors.New("subscription event consumer: record is required")
	}
	if record.Topic != c.topic {
		return fmt.Errorf("subscription event consumer: unexpected topic %q", record.Topic)
	}

	var event subscriptionUpdatedEvent
	if err := json.Unmarshal(record.Value, &event); err != nil {
		return fmt.Errorf("decode Billing subscription event: %w", err)
	}

	eventID, err := uuid.Parse(event.EventID)
	if err != nil {
		return fmt.Errorf("parse Billing event ID: %w", err)
	}

	subscriptionID, err := uuid.Parse(event.SubscriptionID)
	if err != nil {
		return fmt.Errorf("parse Billing subscription ID: %w", err)
	}

	storeID, err := uuid.Parse(event.StoreID)
	if err != nil {
		return fmt.Errorf("parse Billing store ID: %w", err)
	}

	if event.PlanType == "" {
		return errors.New("subscription event consumer: plan type is required")
	}
	if event.Status == "" {
		return errors.New("subscription event consumer: status is required")
	}

	_, err = c.service.Execute(ctx, dto.UpdateAfterSubscriptionInput{
		EventID:        eventID,
		SubscriptionID: subscriptionID,
		StoreID:        storeID,
		PlanType:       event.PlanType,
		Status:         event.Status,
	})
	if err != nil {
		return fmt.Errorf("apply Billing subscription update: %w", err)
	}

	return nil
}

// stop shuts down the polling loop and closes the Kafka client.
func (c *SubscriptionEventConsumer) stop() {
	c.once.Do(func() {
		c.cancel()
		c.client.Close()
		<-c.done
	})
}

// NewSubscriptionEventConsumerForTest creates a consumer for isolated unit tests.
func NewSubscriptionEventConsumerForTest(
	topic string,
	service ports.UpdateAfterSubscriptionService,
) *SubscriptionEventConsumer {
	return &SubscriptionEventConsumer{
		topic:   topic,
		service: service,
	}
}

// HandleRecordForTest exposes record handling for isolated unit tests.
func (c *SubscriptionEventConsumer) HandleRecordForTest(
	ctx context.Context,
	record *kgo.Record,
) error {
	return c.handleRecord(ctx, record)
}
