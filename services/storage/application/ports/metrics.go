package ports

import "context"

// Metric represents a low-cardinality application metric.
//
// Labels must remain bounded and must not contain object references,
// filenames, user IDs, store IDs, or other unbounded values.
type Metric struct {
	Name   string
	Value  float64
	Labels map[string]string
}

// Metrics provides application-level counters and measurements.
type Metrics interface {
	Increment(ctx context.Context, metric Metric) error
	Observe(ctx context.Context, metric Metric) error
}
