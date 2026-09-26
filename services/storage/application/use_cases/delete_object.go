package use_cases

import (
	"context"
	"fmt"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
)

// DeleteObjectUseCase deletes a generic object.
//
// The operation is intentionally generic. The calling service owns the
// object's business meaning and authorization.
type DeleteObjectUseCase struct {
	storage ports.ObjectStorage
	logger  ports.Logger
	metrics ports.Metrics
}

// NewDeleteObjectUseCase constructs the DeleteObject application use case.
func NewDeleteObjectUseCase(
	storage ports.ObjectStorage,
	logger ports.Logger,
	metrics ports.Metrics,
) *DeleteObjectUseCase {
	return &DeleteObjectUseCase{
		storage: storage,
		logger:  logger,
		metrics: metrics,
	}
}

// Execute deletes the requested object.
func (uc *DeleteObjectUseCase) Execute(
	ctx context.Context,
	input dto.DeleteObjectInput,
) error {
	start := time.Now()

	uc.metrics.Increment(ctx, ports.Metric{
		Name:  "storage.delete_object.started",
		Value: 1,
		Labels: map[string]string{
			"operation": "delete_object",
		},
	})

	err := uc.storage.DeleteObject(ctx, input.Reference)
	if err != nil {
		uc.fail(ctx, start, err)
		return err
	}

	uc.logger.Log(ctx, ports.LogEvent{
		Event:     "storage.delete_object.succeeded",
		Operation: "delete_object",
	})

	uc.metrics.Increment(ctx, ports.Metric{
		Name:  "storage.delete_object.succeeded",
		Value: 1,
		Labels: map[string]string{
			"operation": "delete_object",
		},
	})

	uc.metrics.Observe(ctx, ports.Metric{
		Name:  "storage.delete_object.duration",
		Value: float64(time.Since(start).Microseconds()),
		Labels: map[string]string{
			"operation": "delete_object",
		},
	})

	return nil
}

func (uc *DeleteObjectUseCase) fail(
	ctx context.Context,
	start time.Time,
	err error,
) {
	uc.logger.Log(ctx, ports.LogEvent{
		Event:           "storage.delete_object.failed",
		Operation:       "delete_object",
		FailureCategory: fmt.Sprintf("%T", err),
	})

	uc.metrics.Increment(ctx, ports.Metric{
		Name:  "storage.delete_object.failed",
		Value: 1,
		Labels: map[string]string{
			"operation": "delete_object",
		},
	})

	uc.metrics.Observe(ctx, ports.Metric{
		Name:  "storage.delete_object.duration",
		Value: float64(time.Since(start).Microseconds()),
		Labels: map[string]string{
			"operation": "delete_object",
		},
	})
}
