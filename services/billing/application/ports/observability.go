package ports

// Observability groups optional logging and metrics dependencies.
type Observability struct {
	Logger  Logger
	Metrics Metrics
}
