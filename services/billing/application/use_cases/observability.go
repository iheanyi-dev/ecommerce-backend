package use_cases

import (
	"context"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
)

// operationTelemetry records best-effort application telemetry.
// Observability failures never affect business outcomes.
type operationTelemetry struct {
	logger    ports.Logger
	metrics   ports.Metrics
	operation string
	metric    string
}

func newOperationTelemetry(
	logger ports.Logger,
	metrics ports.Metrics,
	operation string,
	metric string,
) operationTelemetry {
	return operationTelemetry{
		logger:    logger,
		metrics:   metrics,
		operation: operation,
		metric:    metric,
	}
}

func (t operationTelemetry) start(ctx context.Context) time.Time {
	startedAt := time.Now()
	t.increment(ctx, t.metric+"_total", nil)
	return startedAt
}

func (t operationTelemetry) success(ctx context.Context, startedAt time.Time) {
	t.log(ctx, ports.LogEvent{
		Event:     "billing." + t.operation + ".succeeded",
		Operation: t.operation,
	})

	t.increment(ctx, t.metric+"_success_total", nil)
	t.observeDuration(ctx, startedAt)
}

func (t operationTelemetry) failure(
	ctx context.Context,
	startedAt time.Time,
	category string,
) {
	t.log(ctx, ports.LogEvent{
		Event:           "billing." + t.operation + ".failed",
		Operation:       t.operation,
		FailureCategory: category,
	})

	t.increment(ctx, t.metric+"_failure_total", map[string]string{
		"failure_category": category,
	})
	t.observeDuration(ctx, startedAt)
}

func (t operationTelemetry) log(ctx context.Context, event ports.LogEvent) {
	if t.logger != nil {
		_ = t.logger.Log(ctx, event)
	}
}

func (t operationTelemetry) increment(
	ctx context.Context,
	name string,
	labels map[string]string,
) {
	if t.metrics == nil {
		return
	}

	_ = t.metrics.Increment(ctx, ports.Metric{
		Name:   name,
		Value:  1,
		Labels: labels,
	})
}

func (t operationTelemetry) observeDuration(ctx context.Context, startedAt time.Time) {
	if t.metrics == nil {
		return
	}

	_ = t.metrics.Observe(ctx, ports.Metric{
		Name:  t.metric + "_duration_seconds",
		Value: time.Since(startedAt).Seconds(),
		Labels: map[string]string{
			"operation": t.operation,
		},
	})
}
