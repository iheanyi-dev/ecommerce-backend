package kafka

import (
	"context"
	"log"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"go.uber.org/fx"
)

type EventPublisher interface {
	Publish(context.Context, ports.SubscriptionOutboxEvent) error
}

type OutboxWorker struct {
	repository ports.SubscriptionOutboxRepository
	producer   EventPublisher
}

func NewOutboxWorker(
	lifecycle fx.Lifecycle,
	repository ports.SubscriptionOutboxRepository,
	producer EventPublisher,
) *OutboxWorker {
	worker := &OutboxWorker{repository: repository, producer: producer}
	var cancel context.CancelFunc
	var done chan struct{}
	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			ctx, stop := context.WithCancel(context.Background())
			cancel, done = stop, make(chan struct{})
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

func (w *OutboxWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
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

// ProcessPending publishes the currently pending outbox events.
func (w *OutboxWorker) ProcessPending(ctx context.Context) {
	w.publishPending(ctx)
}

func (w *OutboxWorker) publishPending(ctx context.Context) {
	events, err := w.repository.GetPending(ctx, 100)
	if err != nil {
		log.Printf("billing outbox: load pending events: %v", err)
		return
	}
	for _, event := range events {
		if ctx.Err() != nil {
			return
		}
		if err := w.producer.Publish(ctx, event); err != nil {
			if recordErr := w.repository.RecordFailure(ctx, event.ID, err); recordErr != nil {
				log.Printf("billing outbox: record failure for %s: %v", event.ID, recordErr)
			}
			log.Printf("billing outbox: publish %s: %v", event.ID, err)
			continue
		}
		if err := w.repository.MarkPublished(ctx, event.ID); err != nil {
			log.Printf("billing outbox: mark %s published: %v", event.ID, err)
		}
	}
}
