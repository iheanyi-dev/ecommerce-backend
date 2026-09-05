package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// AuthenticatedIdentity contains the identity extracted from a valid
// authentication token.
//
// This is intentionally defined at the application boundary rather than
// exposing JWT-specific claims to the Presentation layer.
//
// The application does not need to know whether the underlying token is
// a JWT, opaque token, or another authentication mechanism.
type AuthenticatedIdentity struct {
	UserID string
	Role   string
}

// TokenService defines the application boundary for issuing and validating
// authentication tokens.
//
// The application layer depends on this interface rather than on a concrete
// JWT or token library. Infrastructure provides the implementation.
type TokenService interface {
	// GenerateAccessToken creates an access token for an authenticated user.
	//
	// The token represents the user's identity and authorization role.
	GenerateAccessToken(
		ctx context.Context,
		userID user.UserID,
		role user.Role,
	) (string, error)

	// ValidateAccessToken validates an access token and extracts the
	// authenticated user's identity.
	//
	// The application layer deliberately does not know how the token is
	// encoded or cryptographically verified.
	ValidateAccessToken(
		ctx context.Context,
		token string,
	) (AuthenticatedIdentity, error)
}
