package kafka

import (
	"context"
	"log"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/use_cases"
	"go.uber.org/fx"
)

const (
	expirationWorkerInterval  = time.Hour
	expirationWorkerBatchSize = 100
)

type ExpirationWorker struct {
	expire *use_cases.ExpireSubscriptions
}

func NewExpirationWorker(
	lifecycle fx.Lifecycle,
	expire *use_cases.ExpireSubscriptions,
) *ExpirationWorker {
	worker := &ExpirationWorker{expire: expire}
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

func (w *ExpirationWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(expirationWorkerInterval)
	defer ticker.Stop()

	w.expireDue(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.expireDue(ctx)
		}
	}
}

// ProcessDue expires the currently due subscriptions.
func (w *ExpirationWorker) ProcessDue(ctx context.Context) {
	w.expireDue(ctx)
}

func (w *ExpirationWorker) expireDue(ctx context.Context) {
	if w == nil || w.expire == nil {
		log.Printf("billing expiration worker: use case is unavailable")
		return
	}

	output, err := w.expire.Execute(ctx, use_cases.ExpireSubscriptionsInput{
		Now:       time.Now().UTC(),
		BatchSize: expirationWorkerBatchSize,
	})
	if err != nil {
		log.Printf("billing expiration worker: expire subscriptions: %v", err)
		return
	}

	if output.Expired > 0 {
		log.Printf("billing expiration worker: expired %d subscriptions", output.Expired)
	}
}
