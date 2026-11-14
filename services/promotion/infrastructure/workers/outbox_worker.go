package workers

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/ports"
	"go.uber.org/fx"
)

const (
	outboxPollInterval = 2 * time.Second
	outboxBatchSize    = 100
	outboxClaimLease   = 30 * time.Second
)

type OutboxWorker struct {
	repository ports.PromotionOutboxRepository
	publisher  ports.PromotionEventPublisher
}

func NewOutboxWorker(
	lifecycle fx.Lifecycle,
	repository ports.PromotionOutboxRepository,
	publisher ports.PromotionEventPublisher,
) (*OutboxWorker, error) {
	if repository == nil {
		return nil, errors.New("promotion outbox repository is required")
	}
	if publisher == nil {
		return nil, errors.New("promotion event publisher is required")
	}

	worker := &OutboxWorker{
		repository: repository,
		publisher:  publisher,
	}

	var cancel context.CancelFunc
	var done chan struct{}

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			workerCtx, stop := context.WithCancel(context.Background())
			cancel = stop
			done = make(chan struct{})

			go func() {
				defer close(done)
				worker.run(workerCtx)
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			if cancel == nil {
				return nil
			}

			cancel()

			select {
			case <-done:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	})

	return worker, nil
}

func (w *OutboxWorker) run(ctx context.Context) {
	ticker := time.NewTicker(outboxPollInterval)
	defer ticker.Stop()

	// Process existing events immediately on startup.
	if err := w.ProcessPending(ctx); err != nil && ctx.Err() == nil {
		slog.Error("process promotion outbox", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.ProcessPending(ctx); err != nil && ctx.Err() == nil {
				slog.Error("process promotion outbox", "error", err)
			}
		}
	}
}

func (w *OutboxWorker) ProcessPending(ctx context.Context) error {
	events, err := w.repository.ClaimPending(
		ctx,
		outboxBatchSize,
		outboxClaimLease,
	)
	if err != nil {
		return err
	}

	var processingErrors []error

	for _, event := range events {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(processingErrors, err)...)
		}

		if err := w.publisher.Publish(ctx, event); err != nil {
			if recordErr := w.repository.RecordFailure(
				ctx,
				event.EventID,
				event.ClaimToken,
				err,
			); recordErr != nil {
				processingErrors = append(
					processingErrors,
					errors.Join(err, recordErr),
				)
			}
			continue
		}

		if err := w.repository.MarkPublished(
			ctx,
			event.EventID,
			event.ClaimToken,
		); err != nil {
			processingErrors = append(processingErrors, err)
		}
	}

	return errors.Join(processingErrors...)
}
