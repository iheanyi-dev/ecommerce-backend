package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// UserRepository defines the persistence operations required by the
// Identity application layer.
type UserRepository interface {
	// ExistsByEmail checks whether a user with the supplied email exists.
	ExistsByEmail(ctx context.Context, email user.Email) (bool, error)

	// Create persists a new User aggregate.
	Create(ctx context.Context, newUser *user.User) error

	// FindByEmail retrieves a complete User aggregate by email.
	FindByEmail(ctx context.Context, email user.Email) (*user.User, error)

	// FindByID retrieves a complete User aggregate by ID.
	FindByID(ctx context.Context, id user.UserID) (*user.User, error)

	// UpdateFullName persists a user's changed full name.
	//
	// The email, role, password, and account status are deliberately not
	// included because this operation is limited to profile-name changes.
	UpdateFullName(
		ctx context.Context,
		id user.UserID,
		fullName user.FullName,
	) error
}
