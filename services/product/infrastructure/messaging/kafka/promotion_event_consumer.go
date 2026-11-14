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

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/infrastructure/messaging/events"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/shared/config"
)

const promotionEventTypeHeader = "event_type"

// PromotionEventHandler exposes the Product promotion operations needed by
// the Kafka consumer. The concrete implementation delegates to application
// use cases.
type PromotionEventHandler interface {
	Add(context.Context, use_cases.AddProductPromotionInput) error
	Update(context.Context, use_cases.UpdateProductPromotionInput) error
	Remove(context.Context, use_cases.RemoveProductPromotionInput) error
}

type productPromotionEventHandler struct {
	add    *use_cases.AddProductPromotion
	update *use_cases.UpdateProductPromotion
	remove *use_cases.RemoveProductPromotion
}

// NewPromotionEventHandler adapts Product promotion use cases to the
// consumer's event handler interface.
func NewPromotionEventHandler(
	add *use_cases.AddProductPromotion,
	update *use_cases.UpdateProductPromotion,
	remove *use_cases.RemoveProductPromotion,
) (PromotionEventHandler, error) {
	if add == nil || update == nil || remove == nil {
		return nil, errors.New("product promotion event handler: all use cases are required")
	}

	return &productPromotionEventHandler{
		add:    add,
		update: update,
		remove: remove,
	}, nil
}

func (h *productPromotionEventHandler) Add(
	ctx context.Context,
	input use_cases.AddProductPromotionInput,
) error {
	return h.add.Execute(ctx, input)
}

func (h *productPromotionEventHandler) Update(
	ctx context.Context,
	input use_cases.UpdateProductPromotionInput,
) error {
	return h.update.Execute(ctx, input)
}

func (h *productPromotionEventHandler) Remove(
	ctx context.Context,
	input use_cases.RemoveProductPromotionInput,
) error {
	return h.remove.Execute(ctx, input)
}

// PromotionEventConsumer consumes Promotion lifecycle events and updates
// Product's promotion snapshot index.
type PromotionEventConsumer struct {
	client  *kgo.Client
	handler PromotionEventHandler
	topic   string

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	once   sync.Once
}

// NewPromotionEventConsumer creates the consumer and manages its polling
// goroutine through the Fx lifecycle.
func NewPromotionEventConsumer(
	lifecycle fx.Lifecycle,
	cfg *config.Config,
	handler PromotionEventHandler,
) (*PromotionEventConsumer, error) {
	if cfg == nil {
		return nil, errors.New("promotion event consumer: config is required")
	}
	if cfg.KafkaBrokers == "" {
		return nil, errors.New("promotion event consumer: Kafka brokers are required")
	}
	if cfg.PromotionEventsTopic == "" {
		return nil, errors.New("promotion event consumer: topic is required")
	}
	if cfg.PromotionConsumerGroup == "" {
		return nil, errors.New("promotion event consumer: consumer group is required")
	}
	if handler == nil {
		return nil, errors.New("promotion event consumer: handler is required")
	}

	client, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.KafkaBrokers),
		kgo.ConsumerGroup(cfg.PromotionConsumerGroup),
		kgo.ConsumeTopics(cfg.PromotionEventsTopic),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		return nil, fmt.Errorf("create Product promotion Kafka consumer: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	consumer := &PromotionEventConsumer{
		client:  client,
		handler: handler,
		topic:   cfg.PromotionEventsTopic,
		ctx:     ctx,
		cancel:  cancel,
		done:    make(chan struct{}),
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go consumer.run()
			return nil
		},
		OnStop: func(stopCtx context.Context) error {
			consumer.stop(stopCtx)
			return nil
		},
	})

	return consumer, nil
}

// run processes one record at a time. It does not fetch a later record while
// an earlier record is failing, preventing committed offsets from skipping it.
func (c *PromotionEventConsumer) run() {
	defer close(c.done)

	for {
		fetches := c.client.PollRecords(c.ctx, 1)
		if c.ctx.Err() != nil || fetches.IsClientClosed() {
			return
		}

		for _, fetchErr := range fetches.Errors() {
			slog.Error(
				"Product promotion Kafka fetch failed",
				"topic", fetchErr.Topic,
				"partition", fetchErr.Partition,
				"error", fetchErr.Err,
			)
		}

		fetches.EachRecord(func(record *kgo.Record) {
			for c.ctx.Err() == nil {
				if err := c.handleRecord(c.ctx, record); err != nil {
					slog.Error(
						"Product promotion event processing failed; retrying",
						"topic", record.Topic,
						"partition", record.Partition,
						"offset", record.Offset,
						"error", err,
					)
					if !waitForPromotionRetry(c.ctx) {
						return
					}
					continue
				}
				break
			}

			if c.ctx.Err() != nil {
				return
			}

			// The use case has succeeded. Retry the commit independently so a
			// transient commit failure does not execute the use case again.
			for c.ctx.Err() == nil {
				if err := c.client.CommitRecords(c.ctx, record); err != nil {
					slog.Error(
						"Product promotion offset commit failed; retrying",
						"topic", record.Topic,
						"partition", record.Partition,
						"offset", record.Offset,
						"error", err,
					)
					if !waitForPromotionRetry(c.ctx) {
						return
					}
					continue
				}
				break
			}
		})
	}
}

func waitForPromotionRetry(ctx context.Context) bool {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// HandlePromotionRecord processes one record without starting a Kafka
// consumer. It is also useful for focused consumer tests.
func HandlePromotionRecord(
	ctx context.Context,
	record *kgo.Record,
	handler PromotionEventHandler,
) error {
	return (&PromotionEventConsumer{handler: handler}).handleRecord(ctx, record)
}

func (c *PromotionEventConsumer) handleRecord(
	ctx context.Context,
	record *kgo.Record,
) error {
	if record == nil {
		return errors.New("promotion event consumer: record is required")
	}
	if c.handler == nil {
		return errors.New("promotion event consumer: handler is required")
	}
	if c.topic != "" && record.Topic != c.topic {
		return fmt.Errorf("promotion event consumer: unexpected topic %q", record.Topic)
	}

	eventType := ""
	for _, header := range record.Headers {
		if header.Key == promotionEventTypeHeader {
			eventType = string(header.Value)
			break
		}
	}

	switch eventType {
	case events.PromotionCreated,
		events.PromotionActivated,
		events.PromotionRenewed,
		events.PromotionCancelled,
		events.PromotionExpired:
		// Continue with payload validation and dispatch.
	default:
		// Unknown event types are intentionally ignored and committed. This
		// allows the topic to evolve without blocking Product's consumer.
		slog.Warn(
			"Product promotion consumer ignored unsupported event type",
			"event_type", eventType,
		)
		return nil
	}

	var event events.PromotionEvent
	if err := json.Unmarshal(record.Value, &event); err != nil {
		return fmt.Errorf("decode Promotion event: %w", err)
	}

	if err := validatePromotionEvent(event); err != nil {
		return err
	}

	switch eventType {
	case events.PromotionCreated, events.PromotionRenewed:
		return c.handler.Add(ctx, use_cases.AddProductPromotionInput{
			StoreID:     event.StoreID,
			ProductID:   event.ProductID,
			PromotionID: event.PromoID,
			Status:      event.Status,
		})

	case events.PromotionActivated:
		return c.handler.Update(ctx, use_cases.UpdateProductPromotionInput{
			StoreID:     event.StoreID,
			ProductID:   event.ProductID,
			PromotionID: event.PromoID,
			Status:      event.Status,
		})

	case events.PromotionCancelled, events.PromotionExpired:
		return c.handler.Remove(ctx, use_cases.RemoveProductPromotionInput{
			StoreID:     event.StoreID,
			PromotionID: event.PromoID,
		})
	}

	return nil
}

func validatePromotionEvent(event events.PromotionEvent) error {
	if !validPromotionUUID(event.PromoID) {
		return errors.New("promotion event: promo_id must be a valid UUID")
	}
	if !validPromotionUUID(event.StoreID) {
		return errors.New("promotion event: store_id must be a valid UUID")
	}
	if event.ProductID != nil && !validPromotionUUID(*event.ProductID) {
		return errors.New("promotion event: product_id must be a valid UUID when provided")
	}
	if event.Status == "" {
		return errors.New("promotion event: status is required")
	}
	return nil
}

func validPromotionUUID(value string) bool {
	id, err := uuid.Parse(value)
	return err == nil && id != uuid.Nil
}

// stop cancels polling, closes the Kafka client, and waits for the worker.
func (c *PromotionEventConsumer) stop(ctx context.Context) {
	c.once.Do(func() {
		c.cancel()
		c.client.Close()

		select {
		case <-c.done:
		case <-ctx.Done():
			slog.Warn("Product promotion consumer shutdown timed out", "error", ctx.Err())
		}
	})
}
