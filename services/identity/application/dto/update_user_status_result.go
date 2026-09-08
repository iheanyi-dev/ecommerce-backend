package dto

import "github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"

// UpdateUserStatusResult contains the account information returned after an
// administrator successfully changes a user's status.
type UpdateUserStatusResult struct {
	UserID string
	Status string
}

// NewUpdateUserStatusResult maps the updated domain aggregate into the
// application response DTO.
func NewUpdateUserStatusResult(existingUser *user.User) UpdateUserStatusResult {
	return UpdateUserStatusResult{
		UserID: existingUser.ID().String(),
		Status: string(existingUser.Status()),
	}
}
