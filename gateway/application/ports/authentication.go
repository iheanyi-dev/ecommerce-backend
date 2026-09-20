package ports

import "context"

// AuthenticatedIdentity represents the trusted identity established by the
// Gateway after successfully validating a client access token.
//
// Downstream business services should consume this application-level
// identity rather than parsing or validating JWTs themselves.
//
// The Gateway therefore acts as the external authentication boundary while
// downstream services remain responsible for their own authorization rules.
type AuthenticatedIdentity struct {
	// UserID identifies the authenticated user.
	UserID string

	// Role contains the authorization role carried by the validated token.
	Role string
}

// AccessTokenValidator defines the Gateway's application boundary for
// validating client access tokens.
//
// The Application layer deliberately does not know that the underlying
// implementation uses JWT.
type AccessTokenValidator interface {
	ValidateAccessToken(
		ctx context.Context,
		token string,
	) (AuthenticatedIdentity, error)
}
