package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
)

// CreateStoreService defines the application capability required by the
// presentation layer to create a Store.
//
// The HTTP handler depends on this interface rather than the concrete
// CreateStoreUseCase. This keeps the presentation layer decoupled from the
// application implementation and mirrors the dependency direction used by
// the Identity service.
type CreateStoreService interface {
	Execute(
		ctx context.Context,
		input dto.CreateStoreInput,
	) (dto.CreateStoreOutput, error)
}
