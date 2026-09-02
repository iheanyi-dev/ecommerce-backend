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
	//
	// Registration only needs a boolean answer, so the complete User
	// aggregate is not loaded.
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

	// FindByEmail retrieves a complete User aggregate using its email.
	//
	// Unlike ExistsByEmail, authentication needs the complete persisted
	// user because it must verify the password, account status, role,
	// and eventually use the user's identity when issuing tokens.
	//
	// The infrastructure implementation is responsible for translating
	// raw database values into domain value objects and then rebuilding
	// the aggregate through user.ReconstituteUser().
	FindByEmail(
		ctx context.Context,
		email user.Email,
	) (*user.User, error)
}
