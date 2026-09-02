package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
)

// AuthenticateUserService defines the application boundary for user
// authentication.
//
// Presentation depends on this contract rather than on the concrete
// AuthenticateUserUseCase implementation.
type AuthenticateUserService interface {
	Authenticate(
		ctx context.Context,
		command dto.LoginUserCommand,
	) (dto.LoginUserResult, error)
}
