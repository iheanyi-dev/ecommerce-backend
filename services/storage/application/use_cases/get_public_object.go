package use_cases

import (
	"context"
	"fmt"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
)

// GetPublicObjectUseCase retrieves a public object as a stream.
//
// The object remains streamed through the application boundary. The use case
// does not load the object into memory.
type GetPublicObjectUseCase struct {
	storage ports.ObjectStorage
	logger  ports.Logger
	metrics ports.Metrics
}

// NewGetPublicObjectUseCase constructs the GetPublicObject application
// use case.
func NewGetPublicObjectUseCase(
	storage ports.ObjectStorage,
	logger ports.Logger,
	metrics ports.Metrics,
) *GetPublicObjectUseCase {
	return &GetPublicObjectUseCase{
		storage: storage,
		logger:  logger,
		metrics: metrics,
	}
}

// Execute retrieves the requested public object.
func (uc *GetPublicObjectUseCase) Execute(
	ctx context.Context,
	input dto.GetPublicObjectInput,
) (dto.GetPublicObjectOutput, error) {
	start := time.Now()

	uc.metrics.Increment(ctx, ports.Metric{
		Name:  "storage.get_public_object.started",
		Value: 1,
		Labels: map[string]string{
			"operation": "get_public_object",
		},
	})

	content, contentType, err := uc.storage.GetPublicObject(
		ctx,
		input.Reference,
	)
	if err != nil {
		uc.fail(ctx, start, err)

		return dto.GetPublicObjectOutput{}, err
	}

	uc.logger.Log(ctx, ports.LogEvent{
		Event:     "storage.get_public_object.succeeded",
		Operation: "get_public_object",
	})

	uc.metrics.Increment(ctx, ports.Metric{
		Name:  "storage.get_public_object.succeeded",
		Value: 1,
		Labels: map[string]string{
			"operation": "get_public_object",
		},
	})

	uc.metrics.Observe(ctx, ports.Metric{
		Name:  "storage.get_public_object.duration",
		Value: float64(time.Since(start).Microseconds()),
		Labels: map[string]string{
			"operation": "get_public_object",
		},
	})

	return dto.GetPublicObjectOutput{
		Content:     content,
		ContentType: contentType,
	}, nil
}

func (uc *GetPublicObjectUseCase) fail(
	ctx context.Context,
	start time.Time,
	err error,
) {
	uc.logger.Log(ctx, ports.LogEvent{
		Event:           "storage.get_public_object.failed",
		Operation:       "get_public_object",
		FailureCategory: fmt.Sprintf("%T", err),
	})

	uc.metrics.Increment(ctx, ports.Metric{
		Name:  "storage.get_public_object.failed",
		Value: 1,
		Labels: map[string]string{
			"operation": "get_public_object",
		},
	})

	uc.metrics.Observe(ctx, ports.Metric{
		Name:  "storage.get_public_object.duration",
		Value: float64(time.Since(start).Microseconds()),
		Labels: map[string]string{
			"operation": "get_public_object",
		},
	})
}
