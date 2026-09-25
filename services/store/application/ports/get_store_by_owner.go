package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
)

// GetStoreByOwnerService retrieves the Store belonging to the
// authenticated owner. The owner identity comes from the application
// authentication context rather than caller-supplied input.
type GetStoreByOwnerService interface {
	Execute(ctx context.Context) (*dto.StoreOutput, error)
}
