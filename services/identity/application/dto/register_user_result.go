package dto

import (
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// RegisterUserResult contains the information exposed after successful
// registration.
//
// The result deliberately does not expose the password or password hash.
type RegisterUserResult struct {
	ID        string
	FullName  string
	Email     string
	Role      string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewRegisterUserResult converts the domain User aggregate into an
// application-level result.
//
// This mapping keeps domain objects from leaking into presentation or
// transport layers.
func NewRegisterUserResult(newUser *user.User) RegisterUserResult {
	return RegisterUserResult{
		ID:        newUser.ID().String(),
		FullName:  newUser.FullName().String(),
		Email:     newUser.Email().String(),
		Role:      newUser.Role().String(),
		Status:    newUser.Status().String(),
		CreatedAt: newUser.CreatedAt(),
		UpdatedAt: newUser.UpdatedAt(),
	}
}
