package ports

import "context"

// LogEvent represents a structured Store application observability event.
//
// The event deliberately contains only safe, bounded observability data.
// It must never contain credentials, tokens, passwords, request bodies,
// arbitrary user input, or other sensitive material.
//
// Concrete logging implementations decide how the event is encoded and
// where it is written.
type LogEvent struct {
	// Event identifies the type of event being recorded.
	//
	// Examples:
	//   store.create.succeeded
	//   store.create.failed
	//   store.get_by_id.succeeded
	//   store.get_by_id.failed
	//   store.get_by_slug.succeeded
	//   store.get_by_slug.failed
	Event string

	// Operation identifies the application operation that produced
	// the event.
	//
	// Examples:
	//   create_store
	//   get_store_by_id
	//   get_store_by_slug
	Operation string

	// UserID identifies the authenticated user when safely available.
	//
	// This field must never contain an access token, refresh token,
	// password, email address, or other credential.
	UserID string

	// Role identifies the authenticated user's application role when
	// safely available.
	Role string

	// FailureCategory identifies a safe, high-level failure classification.
	//
	// It must describe the failure without containing raw user input,
	// credentials, database details, or other sensitive information.
	FailureCategory string
}

// Logger defines the application boundary for structured Store logging.
//
// Application code depends on this abstraction rather than a concrete
// logging library. Infrastructure provides the implementation.
//
// Keeping logging behind an application-level port allows tests to use
// lightweight in-memory implementations and keeps logging technology
// outside the business/application layers.
type Logger interface {
	// Log records a structured Store observability event.
	//
	// Logging is an observability concern and must not alter the business
	// result when a concrete implementation encounters an error.
	Log(
		ctx context.Context,
		event LogEvent,
	) error
}
