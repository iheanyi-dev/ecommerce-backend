package ports

// LogEvent describes a structured Cart application event without coupling
// use cases to a concrete logging library.
type LogEvent struct {
	Event           string
	Operation       string
	FailureCategory string
}

// Logger records structured application events.
type Logger interface {
	LogEvent(event LogEvent)
}
