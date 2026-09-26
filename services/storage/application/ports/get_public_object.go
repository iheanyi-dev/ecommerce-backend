package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/dto"
)

// GetPublicObjectService retrieves a publicly visible object as a stream.
//
// Storage remains internally authenticated at the transport boundary.
// "Public" describes the object's business visibility, not direct network
// accessibility of the Storage service.
type GetPublicObjectService interface {
	Execute(
		ctx context.Context,
		input dto.GetPublicObjectInput,
	) (dto.GetPublicObjectOutput, error)
}
