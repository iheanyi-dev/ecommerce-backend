package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/fx"

	notification "github.com/iheanyi-dev/ecommerce-backend/services/notification"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/application/utils"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/domain/entities"
)

type emailExecutor interface {
	Execute(context.Context, entities.Email) error
}

type EmailEventConsumer struct {
	client  *kgo.Client
	useCase emailExecutor
	topic   string

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	once   sync.Once
}

func NewEmailEventConsumer(
	lifecycle fx.Lifecycle,
	cfg *notification.Config,
	useCase *use_cases.SendEmailUseCase,
) (*EmailEventConsumer, error) {
	if cfg == nil {
		return nil, errors.New("notification email consumer: config is required")
	}
	if len(cfg.KafkaBrokers) == 0 {
		return nil, errors.New("notification email consumer: Kafka brokers are required")
	}
	if cfg.NotificationEmailTopic == "" {
		return nil, errors.New("notification email consumer: topic is required")
	}
	if cfg.NotificationEmailConsumer == "" {
		return nil, errors.New("notification email consumer: consumer group is required")
	}
	if useCase == nil {
		return nil, errors.New("notification email consumer: send email use case is required")
	}

	client, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.KafkaBrokers...),
		kgo.ConsumerGroup(cfg.NotificationEmailConsumer),
		kgo.ConsumeTopics(cfg.NotificationEmailTopic),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		return nil, fmt.Errorf("create notification email Kafka client: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	consumer := &EmailEventConsumer{
		client:  client,
		useCase: useCase,
		topic:   cfg.NotificationEmailTopic,
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

func (c *EmailEventConsumer) run() {
	defer close(c.done)

	for {
		fetches := c.client.PollRecords(c.ctx, 1)
		if c.ctx.Err() != nil || fetches.IsClientClosed() {
			return
		}

		for _, fetchErr := range fetches.Errors() {
			slog.Error(
				"Notification email Kafka fetch failed",
				"topic", fetchErr.Topic,
				"partition", fetchErr.Partition,
				"error", fetchErr.Err,
			)
		}

		fetches.EachRecord(func(record *kgo.Record) {
			// Do not advance to another record until this one is handled.
			for c.ctx.Err() == nil {
				if err := c.handleRecord(c.ctx, record); err != nil {
					slog.Error(
						"Notification email delivery failed; retrying",
						"topic", record.Topic,
						"partition", record.Partition,
						"offset", record.Offset,
						"error", err,
					)

					if !waitForRetry(c.ctx) {
						return
					}
					continue
				}
				break
			}

			if c.ctx.Err() != nil {
				return
			}

			// Delivery succeeded or the event was deliberately discarded.
			// Retry the commit independently to avoid sending the email twice.
			for c.ctx.Err() == nil {
				if err := c.client.CommitRecords(c.ctx, record); err != nil {
					slog.Error(
						"Notification email Kafka commit failed; retrying",
						"topic", record.Topic,
						"partition", record.Partition,
						"offset", record.Offset,
						"error", err,
					)

					if !waitForRetry(c.ctx) {
						return
					}
					continue
				}
				break
			}
		})
	}
}

func (c *EmailEventConsumer) handleRecord(
	ctx context.Context,
	record *kgo.Record,
) error {
	if record == nil {
		return errors.New("notification email consumer: record is required")
	}
	if record.Topic != c.topic {
		return fmt.Errorf("notification email consumer: unexpected topic %q", record.Topic)
	}
	if c.useCase == nil {
		return errors.New("notification email consumer: send email use case is unavailable")
	}

	eventType := ""
	for _, header := range record.Headers {
		if header.Key == "event_type" {
			eventType = string(header.Value)
			break
		}
	}

	if eventType == "" {
		slog.Warn("Discarding notification event without event_type header")
		return nil
	}

	email, err := utils.BuildEmail(eventType, record.Value)
	if err != nil {
		// Invalid payloads and unsupported event types are permanent failures.
		// Retrying them indefinitely would block the topic.
		slog.Warn(
			"Discarding unsupported or invalid notification event",
			"event_type", eventType,
			"error", err,
		)
		return nil
	}

	return c.useCase.Execute(ctx, email)
}

func (c *EmailEventConsumer) stop() {
	c.once.Do(func() {
		c.cancel()
		c.client.Close()
		<-c.done
	})
}

func waitForRetry(ctx context.Context) bool {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// NewEmailEventConsumerForTest creates a consumer without a Kafka client.
func NewEmailEventConsumerForTest(
	topic string,
	useCase emailExecutor,
) *EmailEventConsumer {
	return &EmailEventConsumer{
		topic:   topic,
		useCase: useCase,
	}
}

// HandleRecordForTest exposes record handling for isolated unit tests.
func (c *EmailEventConsumer) HandleRecordForTest(
	ctx context.Context,
	record *kgo.Record,
) error {
	return c.handleRecord(ctx, record)
}
