package ports

import "context"

// Metric represents a single Store application measurement.
//
// Metrics are technology-neutral. The application layer does not know
// whether measurements are eventually exported through Prometheus,
// OpenTelemetry, another monitoring backend, or an in-memory
// implementation.
//
// Labels must remain low-cardinality and must never contain sensitive
// or unbounded values such as store IDs, slugs, names, request bodies,
// or credentials.
type Metric struct {
	// Name identifies the metric being recorded.
	Name string

	// Value contains the measurement value.
	//
	// Counter-style metrics may use 1 for each occurrence. Observation
	// metrics may contain values such as operation duration.
	Value float64

	// Labels contain stable, bounded dimensions for the metric.
	//
	// Examples include:
	//   operation
	//   result
	//   failure_category
	//
	// Callers must not use user-controlled identifiers as labels.
	Labels map[string]string
}

// Metrics defines the application boundary for operational Store
// measurements.
//
// Application code depends only on this abstraction. Concrete monitoring
// implementations remain in Infrastructure.
//
// Increment records one occurrence of a counter-style metric.
//
// Observe records a measured numeric value.
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
