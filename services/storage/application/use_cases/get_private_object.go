package use_cases

import (
	"context"
	"fmt"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
)

// GetPrivateObjectUseCase retrieves a private object as a stream.
//
// Storage does not perform business ownership or authorization checks here.
// The trusted internal caller is responsible for establishing authorization
// before invoking the private retrieval operation.
type GetPrivateObjectUseCase struct {
	storage ports.ObjectStorage
	logger  ports.Logger
	metrics ports.Metrics
}

// NewGetPrivateObjectUseCase constructs the GetPrivateObject application
// use case.
func NewGetPrivateObjectUseCase(
	storage ports.ObjectStorage,
	logger ports.Logger,
	metrics ports.Metrics,
) *GetPrivateObjectUseCase {
	return &GetPrivateObjectUseCase{
		storage: storage,
		logger:  logger,
		metrics: metrics,
	}
}

// Execute retrieves the requested private object.
func (uc *GetPrivateObjectUseCase) Execute(
	ctx context.Context,
	input dto.GetPrivateObjectInput,
) (dto.GetPrivateObjectOutput, error) {
	start := time.Now()

	uc.metrics.Increment(ctx, ports.Metric{
		Name:  "storage.get_private_object.started",
		Value: 1,
		Labels: map[string]string{
			"operation": "get_private_object",
		},
	})

	content, contentType, err := uc.storage.GetPrivateObject(
		ctx,
		input.Reference,
	)
	if err != nil {
		uc.fail(ctx, start, err)

		return dto.GetPrivateObjectOutput{}, err
	}

	uc.logger.Log(ctx, ports.LogEvent{
		Event:     "storage.get_private_object.succeeded",
		Operation: "get_private_object",
	})

	uc.metrics.Increment(ctx, ports.Metric{
		Name:  "storage.get_private_object.succeeded",
		Value: 1,
		Labels: map[string]string{
			"operation": "get_private_object",
		},
	})

	uc.metrics.Observe(ctx, ports.Metric{
		Name:  "storage.get_private_object.duration",
		Value: float64(time.Since(start).Microseconds()),
		Labels: map[string]string{
			"operation": "get_private_object",
		},
	})

	return dto.GetPrivateObjectOutput{
		Content:     content,
		ContentType: contentType,
	}, nil
}

func (uc *GetPrivateObjectUseCase) fail(
	ctx context.Context,
	start time.Time,
	err error,
) {
	uc.logger.Log(ctx, ports.LogEvent{
		Event:           "storage.get_private_object.failed",
		Operation:       "get_private_object",
		FailureCategory: fmt.Sprintf("%T", err),
	})

	uc.metrics.Increment(ctx, ports.Metric{
		Name:  "storage.get_private_object.failed",
		Value: 1,
		Labels: map[string]string{
			"operation": "get_private_object",
		},
	})

	uc.metrics.Observe(ctx, ports.Metric{
		Name:  "storage.get_private_object.duration",
		Value: float64(time.Since(start).Microseconds()),
		Labels: map[string]string{
			"operation": "get_private_object",
		},
	})
}
