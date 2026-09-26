package use_cases

import (
	"context"
	"fmt"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
)

// PutObjectUseCase stores an object using the reference supplied by the
// calling service.
//
// Storage intentionally performs no business validation here. The calling
// service is responsible for validating the object's purpose, type, size,
// ownership, and reference before dispatching it to Storage.
type PutObjectUseCase struct {
	storage ports.ObjectStorage
	logger  ports.Logger
	metrics ports.Metrics
}

// NewPutObjectUseCase constructs the PutObject application use case.
func NewPutObjectUseCase(
	storage ports.ObjectStorage,
	logger ports.Logger,
	metrics ports.Metrics,
) *PutObjectUseCase {
	return &PutObjectUseCase{
		storage: storage,
		logger:  logger,
		metrics: metrics,
	}
}

// Execute stores the supplied object.
//
// Observability failures are deliberately ignored so that a logging or
// metrics backend outage cannot change the result of the storage operation.
func (uc *PutObjectUseCase) Execute(
	ctx context.Context,
	input dto.PutObjectInput,
) error {
	start := time.Now()

	uc.metrics.Increment(ctx, ports.Metric{
		Name:  "storage.put_object.started",
		Value: 1,
		Labels: map[string]string{
			"operation": "put_object",
		},
	})

	err := uc.storage.PutObject(
		ctx,
		input.Reference,
		input.Content,
		input.ContentType,
	)
	if err != nil {
		uc.fail(ctx, start, err)
		return err
	}

	uc.logger.Log(ctx, ports.LogEvent{
		Event:     "storage.put_object.succeeded",
		Operation: "put_object",
	})

	uc.metrics.Increment(ctx, ports.Metric{
		Name:  "storage.put_object.succeeded",
		Value: 1,
		Labels: map[string]string{
			"operation": "put_object",
		},
	})

	uc.metrics.Observe(ctx, ports.Metric{
		Name:  "storage.put_object.duration",
		Value: float64(time.Since(start).Microseconds()),
		Labels: map[string]string{
			"operation": "put_object",
		},
	})

	return nil
}

// fail records safe failure observability without changing the original
// storage error returned to the caller.
func (uc *PutObjectUseCase) fail(
	ctx context.Context,
	start time.Time,
	err error,
) {
	uc.logger.Log(ctx, ports.LogEvent{
		Event:           "storage.put_object.failed",
		Operation:       "put_object",
		FailureCategory: fmt.Sprintf("%T", err),
	})

	uc.metrics.Increment(ctx, ports.Metric{
		Name:  "storage.put_object.failed",
		Value: 1,
		Labels: map[string]string{
			"operation": "put_object",
		},
	})

	uc.metrics.Observe(ctx, ports.Metric{
		Name:  "storage.put_object.duration",
		Value: float64(time.Since(start).Microseconds()),
		Labels: map[string]string{
			"operation": "put_object",
		},
	})
}
