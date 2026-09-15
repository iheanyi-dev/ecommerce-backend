package ports

import "context"

// Metric represents a single application or infrastructure measurement.
//
// Metrics are deliberately kept technology-neutral. The application layer
// does not know whether the eventual implementation uses Prometheus,
// OpenTelemetry, another monitoring system, or an in-memory implementation.
//
// Name identifies the metric being recorded.
// Value is used for measurements such as request duration.
// Labels contain low-cardinality dimensions that help distinguish useful
// categories without storing sensitive or unbounded user data.
type Metric struct {
	Name   string
	Value  float64
	Labels map[string]string
}

// Metrics defines the application boundary for operational measurements.
//
// Application and presentation code depend on this abstraction rather than
// directly depending on a monitoring library. Infrastructure provides the
// concrete implementation.
//
// Increment records one occurrence of a counter-style metric.
//
// Observe records a measured numeric value, such as request duration.
type Metrics interface {
	Increment(
		ctx context.Context,
		metric Metric,
	) error

	Observe(
		ctx context.Context,
		metric Metric,
	) error
}
