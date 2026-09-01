package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// UserRepository defines the persistence operations required by the
// Identity application layer.
//
// The application layer depends on this interface rather than on a
// concrete database implementation. The infrastructure layer provides
// the actual persistence implementation.
type UserRepository interface {
	// ExistsByEmail checks whether a user with the supplied email already
	// exists in persistent storage.
	ExistsByEmail(
		ctx context.Context,
		email user.Email,
	) (bool, error)

	// Create persists a new User aggregate.
	//
	// Create is intentionally different from Update. At this point in the
	// application workflow, registration is creating a user that does not
	// already exist.
	Create(
		ctx context.Context,
		newUser *user.User,
	) error
}
