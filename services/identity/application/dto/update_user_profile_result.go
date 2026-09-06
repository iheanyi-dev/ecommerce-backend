package dto

import "github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"

// UpdateUserProfileResult contains the profile information returned after
// a successful profile update.
type UpdateUserProfileResult struct {
	UserID    string
	FullName  string
	Email     string
	Role      string
	Status    string
	UpdatedAt string
}

// NewUpdateUserProfileResult converts the domain User aggregate into the
// application response DTO.
func NewUpdateUserProfileResult(u *user.User) UpdateUserProfileResult {
	return UpdateUserProfileResult{
		UserID:    u.ID().String(),
		FullName:  u.FullName().String(),
		Email:     u.Email().String(),
		Role:      u.Role().String(),
		Status:    u.Status().String(),
		UpdatedAt: u.UpdatedAt().Format("2006-01-02T15:04:05.000Z"),
	}
}
