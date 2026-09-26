package messaging

import (
	"context"
	"sync"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/infrastructure/messaging/events"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/infrastructure/messaging/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/shared/config"
)

const (
	storeImageConsumerGroup         = "storage-store-image"
	storeImageDeletionConsumerGroup = "storage-store-image-deletion"
)

// Consumer consumes Storage-related events published by internal services.
//
// Kafka is deliberately kept inside Infrastructure. The consumer delegates
// actual storage operations to an infrastructure event handler, which in turn
// delegates business work to application use cases. Offsets are committed
// only after successful processing.
type Consumer struct {
	imageClient    *kgo.Client
	deletionClient *kgo.Client
	imageHandler   *handlers.ImageEventHandler

	logger  ports.Logger
	metrics ports.Metrics

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewConsumer constructs the Storage Kafka consumers.
func NewConsumer(
	lc fx.Lifecycle,
	cfg *config.Config,
	putObject ports.PutObjectService,
	deleteObject ports.DeleteObjectService,
	tracer ports.Tracer,
	logger ports.Logger,
	metrics ports.Metrics,
) (*Consumer, error) {
	imageClient, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.KafkaBrokers),
		kgo.ConsumerGroup(storeImageConsumerGroup),
		kgo.ConsumeTopics(events.StoreImageRequestedTopic),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		return nil, err
	}

	deletionClient, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.KafkaBrokers),
		kgo.ConsumerGroup(storeImageDeletionConsumerGroup),
		kgo.ConsumeTopics(events.StoreImageDeletionRequestedTopic),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		imageClient.Close()
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	consumer := &Consumer{
		imageClient:    imageClient,
		deletionClient: deletionClient,
		imageHandler: handlers.NewImageEventHandler(
			putObject,
			deleteObject,
			tracer,
		),
		logger:  logger,
		metrics: metrics,
		cancel:  cancel,
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			consumer.wg.Add(2)

			go consumer.runImageConsumer(ctx)
			go consumer.runDeletionConsumer(ctx)

			return nil
		},

		OnStop: func(ctx context.Context) error {
			cancel()

			imageClient.Close()
			deletionClient.Close()

			done := make(chan struct{})

			go func() {
				consumer.wg.Wait()
				close(done)
			}()

			select {
			case <-done:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	})

	return consumer, nil
}

func (c *Consumer) runImageConsumer(ctx context.Context) {
	defer c.wg.Done()

	for {
		fetches := c.imageClient.PollFetches(ctx)

		if fetches.IsClientClosed() {
			return
		}

		if err := fetches.Err(); err != nil {
			c.recordConsumerFailure(ctx, "store_image", err)
			continue
		}

		stop := false

		fetches.EachRecord(func(record *kgo.Record) {
			if stop {
				return
			}

			if err := c.imageHandler.HandleStoreImage(ctx, record); err != nil {
				c.recordConsumerFailure(ctx, "store_image", err)
				stop = true
				return
			}

			if err := c.imageClient.CommitRecords(ctx, record); err != nil {
				c.recordConsumerFailure(ctx, "store_image_commit", err)
				stop = true
			}
		})
	}
}

func (c *Consumer) runDeletionConsumer(ctx context.Context) {
	defer c.wg.Done()

	for {
		fetches := c.deletionClient.PollFetches(ctx)

		if fetches.IsClientClosed() {
			return
		}

		if err := fetches.Err(); err != nil {
			c.recordConsumerFailure(ctx, "store_image_deletion", err)
			continue
		}

		stop := false

		fetches.EachRecord(func(record *kgo.Record) {
			if stop {
				return
			}

			if err := c.imageHandler.HandleStoreImageDeletion(ctx, record); err != nil {
				c.recordConsumerFailure(ctx, "store_image_deletion", err)
				stop = true
				return
			}

			if err := c.deletionClient.CommitRecords(ctx, record); err != nil {
				c.recordConsumerFailure(ctx, "store_image_deletion_commit", err)
				stop = true
			}
		})
	}
}

func (c *Consumer) recordConsumerFailure(
	ctx context.Context,
	operation string,
	err error,
) {
	_ = c.logger.Log(ctx, ports.LogEvent{
		Event:           "storage.kafka.consumer.failed",
		Operation:       operation,
		FailureCategory: err.Error(),
	})

	_ = c.metrics.Increment(ctx, ports.Metric{
		Name: "storage.kafka.consumer.failed",
		Labels: map[string]string{
			"operation": operation,
		},
	})
}
