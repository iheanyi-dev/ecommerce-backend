package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
)

// RegisterUserService defines the application operation required by the
// presentation layer to register a new user.
//
// The presentation layer depends on this interface instead of the concrete
// RegisterUserUseCase. This keeps HTTP concerns independent from the
// application implementation and makes the handler easy to unit test.
type RegisterUserService interface {
	Execute(
		ctx context.Context,
		command dto.RegisterUserCommand,
	) (dto.RegisterUserResult, error)
}