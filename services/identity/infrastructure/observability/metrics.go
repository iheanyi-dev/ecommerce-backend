package observability

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
)

var (
	// ErrEmptyMetricName indicates that a metric was submitted without a
	// name. A name is required so measurements can be identified reliably.
	ErrEmptyMetricName = errors.New("metric name cannot be empty")
)

// Metrics is the Infrastructure implementation of the application-level
// ports.Metrics contract.
//
// This implementation intentionally keeps measurements in memory. It gives
// the Identity service a concrete metrics boundary without coupling the
// application layer to Prometheus, OpenTelemetry, or another monitoring
// system.
//
// A production monitoring backend can be introduced later behind the same
// application-level contract.
type Metrics struct {
	mu sync.RWMutex

	counters     map[string]float64
	observations map[string][]float64
}

// NewMetrics creates an in-memory metrics collector.
//
// The collector is safe for concurrent use because HTTP requests and
// application operations may record metrics concurrently.
func NewMetrics() *Metrics {
	return &Metrics{
		counters:     make(map[string]float64),
		observations: make(map[string][]float64),
	}
}

// Increment records one occurrence of a counter-style metric.
//
// Labels are part of the metric identity. This allows useful low-cardinality
// dimensions such as HTTP method, route, and status code without combining
// unrelated measurements into one counter.
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

// Observe records a numeric measurement such as request duration.
//
// Observations are retained in memory because this implementation is intended
// to provide a concrete, testable metrics boundary before a dedicated
// monitoring backend is selected.
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
// This method exists for infrastructure-level verification and diagnostics.
// Application code should record metrics through ports.Metrics rather than
// depending on this concrete implementation.
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

// ObservationValues returns the recorded observations for a metric.
//
// A copy is returned so callers cannot mutate the collector's internal
// state.
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

	values := m.observations[key]

	return append([]float64(nil), values...)
}

// metricKey creates a stable internal key from a metric name and its labels.
//
// Labels are sorted before constructing the key so that equivalent label maps
// produce the same metric identity regardless of map iteration order.
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
