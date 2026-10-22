package ports

// Metric describes a Cart application metric.
type Metric struct {
	Name   string
	Value  float64
	Labels map[string]string
}

// Metrics records application-level measurements.
type Metrics interface {
	Record(metric Metric)
}
