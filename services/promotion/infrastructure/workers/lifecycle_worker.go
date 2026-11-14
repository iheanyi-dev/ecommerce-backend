package workers

import (
	"context"
	"log"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/dtos"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/use_cases"
	"go.uber.org/fx"
)

const (
	lifecycleWorkerInterval  = time.Minute
	lifecycleWorkerBatchSize = 100
)

type promotionActivator interface {
	Execute(context.Context, dtos.ActivatePromotionInput) (dtos.PromotionDTO, error)
}

type promotionExpirer interface {
	Execute(context.Context, dtos.ExpirePromotionInput) (dtos.PromotionDTO, error)
}

type duePromotionRepository interface {
	ListScheduledDue(context.Context, time.Time, int) ([]string, error)
	ListExpiredDue(context.Context, time.Time, int) ([]string, error)
}

type LifecycleWorker struct {
	repository duePromotionRepository
	activate   promotionActivator
	expire     promotionExpirer
}

func NewLifecycleWorker(
	lifecycle fx.Lifecycle,
	repository duePromotionRepository,
	activate *use_cases.ActivatePromotionUseCase,
	expire *use_cases.ExpirePromotionUseCase,
) *LifecycleWorker {
	worker := NewLifecycleWorkerRunner(repository, activate, expire)

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

func NewLifecycleWorkerRunner(
	repository duePromotionRepository,
	activate promotionActivator,
	expire promotionExpirer,
) *LifecycleWorker {
	return &LifecycleWorker{
		repository: repository,
		activate:   activate,
		expire:     expire,
	}
}

func (w *LifecycleWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(lifecycleWorkerInterval)
	defer ticker.Stop()

	w.ProcessDue(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.ProcessDue(ctx)
		}
	}
}

// ProcessDue activates scheduled promotions and expires ended promotions.
func (w *LifecycleWorker) ProcessDue(ctx context.Context) {
	if w == nil || w.repository == nil {
		log.Printf("promotion lifecycle worker: repository is unavailable")
		return
	}

	now := time.Now().UTC()
	w.activateDue(ctx, now)
	w.expireDue(ctx, now)
}

func (w *LifecycleWorker) activateDue(ctx context.Context, now time.Time) {
	if w.activate == nil {
		log.Printf("promotion lifecycle worker: activation use case is unavailable")
		return
	}

	ids, err := w.repository.ListScheduledDue(ctx, now, lifecycleWorkerBatchSize)
	if err != nil {
		log.Printf("promotion lifecycle worker: list scheduled promotions: %v", err)
		return
	}

	activated := 0
	for _, id := range ids {
		if ctx.Err() != nil {
			return
		}
		if _, err := w.activate.Execute(ctx, dtos.ActivatePromotionInput{
			PromotionID: id,
		}); err != nil {
			log.Printf("promotion lifecycle worker: activate promotion %s: %v", id, err)
			continue
		}
		activated++
	}

	if activated > 0 {
		log.Printf("promotion lifecycle worker: activated %d promotions", activated)
	}
}

func (w *LifecycleWorker) expireDue(ctx context.Context, now time.Time) {
	if w.expire == nil {
		log.Printf("promotion lifecycle worker: expiration use case is unavailable")
		return
	}

	ids, err := w.repository.ListExpiredDue(ctx, now, lifecycleWorkerBatchSize)
	if err != nil {
		log.Printf("promotion lifecycle worker: list expired promotions: %v", err)
		return
	}

	expired := 0
	for _, id := range ids {
		if ctx.Err() != nil {
			return
		}
		if _, err := w.expire.Execute(ctx, dtos.ExpirePromotionInput{
			PromotionID: id,
		}); err != nil {
			log.Printf("promotion lifecycle worker: expire promotion %s: %v", id, err)
			continue
		}
		expired++
	}

	if expired > 0 {
		log.Printf("promotion lifecycle worker: expired %d promotions", expired)
	}
}
