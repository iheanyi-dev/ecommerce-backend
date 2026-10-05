package use_cases

import (
	"context"
	"fmt"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
)

// HeadObjectUseCase checks for an object and returns its content type.
//
// This is intentionally metadata-only. It does not open or stream the object
// body, which makes it suitable for existence checks and startup provisioning
// without unnecessary I/O.
type HeadObjectUseCase struct {
	storage ports.ObjectStorage
	logger  ports.Logger
	metrics ports.Metrics
}

// NewHeadObjectUseCase constructs the HeadObject application use case.
func NewHeadObjectUseCase(
	storage ports.ObjectStorage,
	logger ports.Logger,
	metrics ports.Metrics,
) *HeadObjectUseCase {
	return &HeadObjectUseCase{
		storage: storage,
		logger:  logger,
		metrics: metrics,
	}
}

// Execute checks whether the requested object exists.
func (uc *HeadObjectUseCase) Execute(
	ctx context.Context,
	input dto.HeadObjectInput,
) (dto.HeadObjectOutput, error) {
	start := time.Now()

	uc.metrics.Increment(ctx, ports.Metric{
		Name:  "storage.head_object.started",
		Value: 1,
		Labels: map[string]string{
			"operation": "head_object",
		},
	})

	contentType, err := uc.storage.HeadObject(
		ctx,
		input.Reference,
	)
	if err != nil {
		uc.fail(ctx, start, err)

		return dto.HeadObjectOutput{}, err
	}

	uc.logger.Log(ctx, ports.LogEvent{
		Event:     "storage.head_object.succeeded",
		Operation: "head_object",
	})

	uc.metrics.Increment(ctx, ports.Metric{
		Name:  "storage.head_object.succeeded",
		Value: 1,
		Labels: map[string]string{
			"operation": "head_object",
		},
	})

	uc.metrics.Observe(ctx, ports.Metric{
		Name:  "storage.head_object.duration",
		Value: float64(time.Since(start).Microseconds()),
		Labels: map[string]string{
			"operation": "head_object",
		},
	})

	return dto.HeadObjectOutput{
		ContentType: contentType,
	}, nil
}

func (uc *HeadObjectUseCase) fail(
	ctx context.Context,
	start time.Time,
	err error,
) {
	uc.logger.Log(ctx, ports.LogEvent{
		Event:           "storage.head_object.failed",
		Operation:       "head_object",
		FailureCategory: fmt.Sprintf("%T", err),
	})

	uc.metrics.Increment(ctx, ports.Metric{
		Name:  "storage.head_object.failed",
		Value: 1,
		Labels: map[string]string{
			"operation": "head_object",
		},
	})

	uc.metrics.Observe(ctx, ports.Metric{
		Name:  "storage.head_object.duration",
		Value: float64(time.Since(start).Microseconds()),
		Labels: map[string]string{
			"operation": "head_object",
		},
	})
}
