package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
)

// UpdateUserProfileService defines the application operation used to update
// the authenticated user's mutable profile information.
type UpdateUserProfileService interface {
	Execute(
		ctx context.Context,
		userID string,
		command dto.UpdateUserProfileCommand,
	) (dto.UpdateUserProfileResult, error)
}
