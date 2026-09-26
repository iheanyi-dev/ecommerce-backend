package observability

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
)

var (
	// ErrEmptyMetricName indicates that a metric was submitted without a
	// name.
	ErrEmptyMetricName = errors.New("metric name cannot be empty")
)

// Metrics is the Infrastructure implementation of the application-level
// ports.Metrics contract.
//
// The implementation is intentionally in-memory. The application layer
// remains independent of Prometheus, OpenTelemetry, or another monitoring
// backend.
type Metrics struct {
	mu sync.RWMutex

	counters     map[string]float64
	observations map[string][]float64
}

// NewMetrics creates a concurrent-safe in-memory metrics collector.
func NewMetrics() *Metrics {
	return &Metrics{
		counters:     make(map[string]float64),
		observations: make(map[string][]float64),
	}
}

// Increment records one occurrence of a counter-style metric.
func (m *Metrics) Increment(
	ctx context.Context,
	metric ports.Metric,
) error {
	if m == nil {
		return errors.New("metrics is not initialized")
	}

	if strings.TrimSpace(metric.Name) == "" {
		return ErrEmptyMetricName
	}

	key := metricKey(metric.Name, metric.Labels)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters[key]++

	return nil
}

// Observe records a numeric measurement such as operation duration.
func (m *Metrics) Observe(
	ctx context.Context,
	metric ports.Metric,
) error {
	if m == nil {
		return errors.New("metrics is not initialized")
	}

	if strings.TrimSpace(metric.Name) == "" {
		return ErrEmptyMetricName
	}

	key := metricKey(metric.Name, metric.Labels)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.observations[key] = append(
		m.observations[key],
		metric.Value,
	)

	return nil
}

// CounterValue returns the current value of a counter.
//
// This is intended for infrastructure-level verification and diagnostics.
// Application code depends only on ports.Metrics.
func (m *Metrics) CounterValue(
	name string,
	labels ...map[string]string,
) float64 {
	if m == nil {
		return 0
	}

	var metricLabels map[string]string

	if len(labels) > 0 {
		metricLabels = labels[0]
	}

	key := metricKey(name, metricLabels)

	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.counters[key]
}

// ObservationValues returns a copy of the observations recorded for a metric.
func (m *Metrics) ObservationValues(
	name string,
	labels ...map[string]string,
) []float64 {
	if m == nil {
		return nil
	}

	var metricLabels map[string]string

	if len(labels) > 0 {
		metricLabels = labels[0]
	}

	key := metricKey(name, metricLabels)

	m.mu.RLock()
	defer m.mu.RUnlock()

	return append([]float64(nil), m.observations[key]...)
}

// metricKey creates a stable internal key from a metric name and labels.
//
// Sorting the labels makes equivalent maps produce the same metric identity.
func metricKey(
	name string,
	labels map[string]string,
) string {
	if len(labels) == 0 {
		return name
	}

	keys := make([]string, 0, len(labels))

	for key := range labels {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	var builder strings.Builder

	builder.WriteString(name)

	for _, key := range keys {
		builder.WriteByte('|')
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(labels[key])
	}

	return builder.String()
}
