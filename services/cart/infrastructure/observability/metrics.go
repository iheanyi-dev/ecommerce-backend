package observability

import (
	"sort"
	"strings"
	"sync"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/ports"
)

// Metrics is a concurrency-safe, in-memory collector for Cart metrics.
// Each Record call stores the supplied value as an observation.
type Metrics struct {
	mu           sync.RWMutex
	observations map[string][]float64
}

var _ ports.Metrics = (*Metrics)(nil)

// NewMetrics creates an empty in-memory metrics collector.
func NewMetrics() *Metrics {
	return &Metrics{
		observations: make(map[string][]float64),
	}
}

// Record stores the supplied metric value and groups it by name and labels.
// Invalid metrics and calls on a nil receiver are ignored because the
// application port does not return errors.
func (m *Metrics) Record(metric ports.Metric) {
	if m == nil || strings.TrimSpace(metric.Name) == "" {
		return
	}

	key := metricKey(metric.Name, metric.Labels)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.observations[key] = append(m.observations[key], metric.Value)
}

// ObservationValues returns a copy of the recorded values for a metric.
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

// metricKey produces a deterministic key regardless of label map iteration order.
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
