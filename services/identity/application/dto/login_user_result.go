package dto

import "github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"

// LoginUserResult contains the authenticated user's identity and
// authentication token.
//
// Password information is deliberately excluded.
type LoginUserResult struct {
	ID           string
	Email        string
	Role         string
	Status       string
	AccessToken  string
}

// NewLoginUserResult converts the authenticated User aggregate into
// an application-level result.
//
// The access token is supplied separately because it is generated only
// after successful authentication.
func NewLoginUserResult(
	authenticatedUser *user.User,
	accessToken string,
) LoginUserResult {
	return LoginUserResult{
		ID:          authenticatedUser.ID().String(),
		Email:       authenticatedUser.Email().String(),
		Role:        authenticatedUser.Role().String(),
		Status:      authenticatedUser.Status().String(),
		AccessToken: accessToken,
	}
}