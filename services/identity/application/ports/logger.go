package ports

import "context"

// LogEvent represents a structured application event.
//
// The event contains only observability information. It deliberately does
// not contain credentials, tokens, passwords, password hashes, or other
// authentication secrets.
//
// Concrete logging implementations are responsible for deciding how the
// event is encoded and where it is written.
type LogEvent struct {
	// Event identifies the type of event being recorded.
	//
	// Examples:
	//   auth.login.succeeded
	//   auth.login.failed
	//   authorization.denied
	//   http.request.completed
	Event string

	// Operation identifies the operation that produced the event.
	//
	// Examples:
	//   login
	//   logout
	//   refresh
	//   register
	Operation string

	// UserID identifies the authenticated user when one is safely available.
	//
	// This must never contain an access token, refresh token, password,
	// email address, or other credential.
	UserID string

	// Role identifies the user's application role when safely available.
	Role string

	// HTTPMethod contains the HTTP method associated with the event when
	// applicable.
	HTTPMethod string

	// Route contains the normalized application route when applicable.
	//
	// Callers should provide the route pattern rather than sensitive
	// request data or query parameters.
	Route string

	// StatusCode contains the HTTP response status when applicable.
	StatusCode int

	// DurationMillis contains the operation duration in milliseconds when
	// applicable.
	DurationMillis int64

	// FailureCategory identifies a safe, high-level failure classification.
	//
	// It should describe the reason for failure without containing secrets
	// or user-provided credential material.
	FailureCategory string

	// RequiredRole identifies the role required by an authorization check
	// when applicable.
	RequiredRole string
}

// Logger defines the application boundary for structured observability.
//
// Application and presentation code depend on this abstraction rather than
// on a concrete logging library such as Zap. Infrastructure provides the
// implementation.
//
// Keeping logging behind an application-level port also allows tests to use
// lightweight in-memory implementations without requiring a real logger.
type Logger interface {
	// Log records a structured observability event.
	//
	// Implementations should respect the supplied context and should not
	// expose sensitive authentication material.
	Log(
		ctx context.Context,
		event LogEvent,
	) error
}
