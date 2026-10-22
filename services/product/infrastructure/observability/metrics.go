package observability

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
)

var (
	// ErrEmptyMetricName indicates a missing or blank metric name.
	ErrEmptyMetricName = errors.New("metric name cannot be empty")
)

// Metrics is a concurrency-safe, in-memory implementation of ports.Metrics.
// A future exporter can replace this adapter without changing use cases.
type Metrics struct {
	mu           sync.RWMutex
	counters     map[string]float64
	observations map[string][]float64
}

var _ ports.Metrics = (*Metrics)(nil)

// NewMetrics creates an empty metrics collector.
func NewMetrics() *Metrics {
	return &Metrics{
		counters:     make(map[string]float64),
		observations: make(map[string][]float64),
	}
}

// Increment records one occurrence, independent of metric.Value.
func (m *Metrics) Increment(_ context.Context, metric ports.Metric) error {
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

// Observe records one numeric observation.
func (m *Metrics) Observe(_ context.Context, metric ports.Metric) error {
	if m == nil {
		return errors.New("metrics is not initialized")
	}
	if strings.TrimSpace(metric.Name) == "" {
		return ErrEmptyMetricName
	}

	key := metricKey(metric.Name, metric.Labels)

	m.mu.Lock()
	defer m.mu.Unlock()
	m.observations[key] = append(m.observations[key], metric.Value)

	return nil
}

// CounterValue returns a counter value for infrastructure tests.
func (m *Metrics) CounterValue(name string, labels ...map[string]string) float64 {
	if m == nil {
		return 0
	}

	var metricLabels map[string]string
	if len(labels) > 0 {
		metricLabels = labels[0]
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.counters[metricKey(name, metricLabels)]
}

// ObservationValues returns a copy of the recorded observations.
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

	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]float64(nil), m.observations[metricKey(name, metricLabels)]...)
}

// metricKey builds a deterministic identity from a name and its labels.
func metricKey(name string, labels map[string]string) string {
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
