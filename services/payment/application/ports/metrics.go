package ports

import "context"

// Metric represents a low-cardinality application measurement.
// Never use payment IDs, payer IDs, or business references as labels.
type Metric struct {
	Name   string
	Value  float64
	Labels map[string]string
}

// Metrics records application measurements. Metrics failures must not
// change the business operation's result.
type Metrics interface {
	Increment(ctx context.Context, metric Metric) error
	Observe(ctx context.Context, metric Metric) error
}
