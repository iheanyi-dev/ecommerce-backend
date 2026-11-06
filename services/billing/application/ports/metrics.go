package ports

import "context"

// Metric represents a bounded application metric.
type Metric struct {
	Name   string
	Value  float64
	Labels map[string]string
}

// Metrics records counters and observations.
// Implementations must not be allowed to affect business outcomes.
type Metrics interface {
	Increment(ctx context.Context, metric Metric) error
	Observe(ctx context.Context, metric Metric) error
}
