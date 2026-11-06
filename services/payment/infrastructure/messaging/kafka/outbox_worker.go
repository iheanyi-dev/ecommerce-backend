package kafka

import (
	"context"
	"log"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"go.uber.org/fx"
)

const (
	outboxBatchSize = 100
	outboxPollDelay = 2 * time.Second
)

// eventPublisher publishes an outbox event to the message broker.
type eventPublisher interface {
	Publish(context.Context, ports.PaymentOutboxEvent) error
}

// OutboxWorker delivers persisted payment events to Kafka.
type OutboxWorker struct {
	repository ports.PaymentOutboxRepository
	producer   eventPublisher
}

// NewOutboxWorker creates the payment outbox worker.
func NewOutboxWorker(
	lifecycle fx.Lifecycle,
	repository ports.PaymentOutboxRepository,
	producer eventPublisher,
) *OutboxWorker {
	worker := &OutboxWorker{
		repository: repository,
		producer:   producer,
	}

	var cancel context.CancelFunc
	var done chan struct{}

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			ctx, stop := context.WithCancel(context.Background())
			cancel = stop
			done = make(chan struct{})

			go func() {
				defer close(done)
				worker.Run(ctx)
			}()
			return nil
		},
		OnStop: func(context.Context) error {
			if cancel != nil {
				cancel()
				<-done
			}
			return nil
		},
	})

	return worker
}

// Run polls PostgreSQL and publishes pending events until the context is cancelled.
func (w *OutboxWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(outboxPollDelay)
	defer ticker.Stop()

	w.publishPending(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.publishPending(ctx)
		}
	}
}

func (w *OutboxWorker) publishPending(ctx context.Context) {
	events, err := w.repository.GetPending(ctx, outboxBatchSize)
	if err != nil {
		log.Printf("payment outbox: load pending events: %v", err)
		return
	}

	for _, event := range events {
		if ctx.Err() != nil {
			return
		}

		if err := w.producer.Publish(ctx, event); err != nil {
			if recordErr := w.repository.RecordFailure(ctx, event.ID, err); recordErr != nil {
				log.Printf("payment outbox: record delivery failure for %s: %v", event.ID, recordErr)
			}
			log.Printf("payment outbox: publish event %s: %v", event.ID, err)
			continue
		}

		if err := w.repository.MarkPublished(ctx, event.ID); err != nil {
			log.Printf("payment outbox: mark event %s published: %v", event.ID, err)
		}
	}
}
