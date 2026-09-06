package ports

import "context"

// ChangePasswordService defines the application boundary for changing
// the password of an authenticated user.
//
// The application layer owns the password-change workflow while keeping
// HTTP, JWT, bcrypt, and database-specific details outside the use case.
type ChangePasswordService interface {
	Execute(
		ctx context.Context,
		userID string,
		currentPassword string,
		newPassword string,
	) error
}
