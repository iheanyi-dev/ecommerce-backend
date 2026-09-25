package observability

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
)

var (
	// ErrEmptyMetricName indicates that a metric was submitted without a
	// usable metric name.
	ErrEmptyMetricName = errors.New("metric name cannot be empty")
)

// Metrics is the Infrastructure implementation of the Store
// application-level ports.Metrics contract.
//
// The implementation intentionally remains technology-neutral and in-memory.
// A production exporter can later be placed behind the same application
// contract without changing Store use cases.
type Metrics struct {
	mu sync.RWMutex

	counters     map[string]float64
	observations map[string][]float64
}

// Compile-time contract verification.
var _ ports.Metrics = (*Metrics)(nil)

// NewMetrics creates a concurrency-safe in-memory metrics collector.
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

// Observe records a numeric observation such as an operation duration.
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
// This is intentionally an infrastructure-level inspection method.
// Application code records measurements through ports.Metrics.
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

// ObservationValues returns a copy of all recorded observations for a metric.
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

// metricKey creates a deterministic internal identity from a metric name
// and its low-cardinality labels.
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
