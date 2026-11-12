package use_cases

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
)

func recordProductPromotionOperation(
	ctx context.Context,
	logger ports.Logger,
	metrics ports.Metrics,
	operation string,
	event string,
	failureCategory string,
) {
	if logger != nil {
		_ = logger.Log(ctx, ports.LogEvent{
			Event:           event,
			Operation:       operation,
			FailureCategory: failureCategory,
		})
	}

	if metrics != nil {
		_ = metrics.Increment(ctx, ports.Metric{
			Name:  "product_operations_total",
			Value: 1,
			Labels: map[string]string{
				"operation": operation,
				"event":     event,
			},
		})
	}
}
