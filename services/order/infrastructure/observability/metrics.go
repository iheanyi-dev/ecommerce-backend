package observability

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
)

var (
	// ErrEmptyMetricName indicates that a metric has no usable name.
	ErrEmptyMetricName = errors.New("metric name cannot be empty")
)

// Metrics is a concurrency-safe, in-memory implementation of the Order
// Metrics port.
//
// This implementation is suitable as an initial adapter and for tests.
// It can later be replaced with a Prometheus or OpenTelemetry adapter without
// changing application-layer code.
type Metrics struct {
	mu           sync.RWMutex
	counters     map[string]float64
	observations map[string][]float64
}

// NewMetrics creates an empty in-memory metrics registry.
func NewMetrics() *Metrics {
	return &Metrics{
		counters:     make(map[string]float64),
		observations: make(map[string][]float64),
	}
}

// Increment adds the metric value to the counter identified by its name
// and labels. An empty name is rejected rather than silently recorded.
func (m *Metrics) Increment(_ context.Context, metric ports.Metric) error {
	if m == nil {
		return errors.New("metrics cannot be nil")
	}

	key, err := metricKey(metric)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters[key] += metric.Value

	return nil
}

// Observe records one observation for the metric name and labels.
func (m *Metrics) Observe(_ context.Context, metric ports.Metric) error {
	if m == nil {
		return errors.New("metrics cannot be nil")
	}

	key, err := metricKey(metric)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.observations[key] = append(m.observations[key], metric.Value)

	return nil
}

// CounterValue returns the counter value for the supplied metric name and
// labels. A counter with no recorded increments returns zero.
func (m *Metrics) CounterValue(name string, labels map[string]string) float64 {
	if m == nil {
		return 0
	}

	key, err := metricKey(ports.Metric{Name: name, Labels: labels})
	if err != nil {
		return 0
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.counters[key]
}

// ObservationValues returns a copy of the recorded observations for the
// supplied metric name and labels. Callers cannot mutate internal state.
func (m *Metrics) ObservationValues(
	name string,
	labels map[string]string,
) []float64 {
	if m == nil {
		return nil
	}

	key, err := metricKey(ports.Metric{Name: name, Labels: labels})
	if err != nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	values := m.observations[key]
	if len(values) == 0 {
		return []float64{}
	}

	return append([]float64(nil), values...)
}

// metricKey creates a stable key from a metric name and its labels.
//
// Sorting label names ensures that equivalent label maps produce the same
// key regardless of Go map iteration order. Length-prefixed components avoid
// collisions caused by separators appearing in label values.
func metricKey(metric ports.Metric) (string, error) {
	if strings.TrimSpace(metric.Name) == "" {
		return "", ErrEmptyMetricName
	}

	var key strings.Builder
	writeKeyPart(&key, metric.Name)

	labelNames := make([]string, 0, len(metric.Labels))
	for name := range metric.Labels {
		labelNames = append(labelNames, name)
	}
	sort.Strings(labelNames)

	for _, name := range labelNames {
		writeKeyPart(&key, name)
		writeKeyPart(&key, metric.Labels[name])
	}

	return key.String(), nil
}

// writeKeyPart adds an unambiguous, length-prefixed component to a metric key.
func writeKeyPart(builder *strings.Builder, value string) {
	builder.WriteString(stringInt(len(value)))
	builder.WriteByte(':')
	builder.WriteString(value)
	builder.WriteByte('|')
}

// stringInt converts a nonnegative integer to decimal without introducing
// another formatting dependency into the metric-key helper.
func stringInt(value int) string {
	if value == 0 {
		return "0"
	}

	var digits [20]byte
	position := len(digits)

	for value > 0 {
		position--
		digits[position] = byte('0' + value%10)
		value /= 10
	}

	return string(digits[position:])
}

// Ensure Metrics implements the application-layer metrics contract.
var _ ports.Metrics = (*Metrics)(nil)
