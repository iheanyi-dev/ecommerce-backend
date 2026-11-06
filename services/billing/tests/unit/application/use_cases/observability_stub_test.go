package use_cases_test

import (
	"context"
	"errors"

	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
)

type recordingLogger struct {
	events []ports.LogEvent
	err    error
}

func (logger *recordingLogger) Log(
	_ context.Context,
	event ports.LogEvent,
) error {
	logger.events = append(logger.events, event)
	return logger.err
}

type recordingMetrics struct {
	increments   []ports.Metric
	observations []ports.Metric
	incrementErr error
	observeErr   error
}

func (metrics *recordingMetrics) Increment(
	_ context.Context,
	metric ports.Metric,
) error {
	metrics.increments = append(metrics.increments, metric)
	return metrics.incrementErr
}

func (metrics *recordingMetrics) Observe(
	_ context.Context,
	metric ports.Metric,
) error {
	metrics.observations = append(metrics.observations, metric)
	return metrics.observeErr
}

func newRecordingObservability() (
	*recordingLogger,
	*recordingMetrics,
	ports.Observability,
) {
	logger := &recordingLogger{}
	metrics := &recordingMetrics{}

	return logger, metrics, ports.Observability{
		Logger:  logger,
		Metrics: metrics,
	}
}

func newFailingObservability() ports.Observability {
	return ports.Observability{
		Logger: &recordingLogger{err: errors.New("logging unavailable")},
		Metrics: &recordingMetrics{
			incrementErr: errors.New("metrics unavailable"),
			observeErr:   errors.New("metrics unavailable"),
		},
	}
}
