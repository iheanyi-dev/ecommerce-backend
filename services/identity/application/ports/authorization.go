package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// AuthorizationService determines whether an authenticated identity
// is allowed to perform an operation.
//
// Authentication answers:
//
//	"Who is this user?"
//
// Authorization answers:
//
//	"Is this user allowed to perform this operation?"
//
// Keeping this contract in the application layer prevents the
// presentation layer from becoming coupled to authorization rules.
type AuthorizationService interface {
	Authorize(
		ctx context.Context,
		userID user.UserID,
		role user.Role,
		resource string,
		action string,
	) error
}
