package ports

import "context"

type Metric struct {
	Name   string
	Value  float64
	Labels map[string]string
}

type Metrics interface {
	Increment(ctx context.Context, metric Metric) error
	Observe(ctx context.Context, metric Metric) error
}
