package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
)

// RefreshUserService defines the application boundary for refreshing
// an authenticated user's access and refresh tokens.
//
// Presentation depends on this contract rather than on the concrete
// RefreshUserUseCase implementation.
type RefreshUserService interface {
	Refresh(
		ctx context.Context,
		command dto.RefreshTokenCommand,
	) (dto.RefreshTokenResult, error)
}
