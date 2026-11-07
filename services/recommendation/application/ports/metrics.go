package ports

import "context"

// Metric represents a Recommendation application measurement.
type Metric struct {
	Name   string
	Value  float64
	Labels map[string]string
}

// Metrics defines the application boundary for operational measurements.
type Metrics interface {
	Increment(ctx context.Context, metric Metric) error
	Observe(ctx context.Context, metric Metric) error
}
