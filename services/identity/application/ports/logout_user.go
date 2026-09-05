package ports

import "context"

// LogoutUserService defines the application boundary for logging out
// the current refresh-token session.
//
// The presentation layer depends on this interface rather than on the
// concrete LogoutUserUseCase implementation. This keeps the application
// layer independent from HTTP and other delivery mechanisms.
type LogoutUserService interface {
	// Logout revokes only the refresh-token session represented by
	// the supplied raw refresh token.
	Logout(ctx context.Context, refreshToken string) error
}
