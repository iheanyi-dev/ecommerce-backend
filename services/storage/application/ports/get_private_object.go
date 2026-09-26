package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/dto"
)

// GetPrivateObjectService retrieves a private object as a stream.
//
// Storage performs no business-level authorization. The trusted internal
// caller owns the authorization decision before invoking this operation.
type GetPrivateObjectService interface {
	Execute(
		ctx context.Context,
		input dto.GetPrivateObjectInput,
	) (dto.GetPrivateObjectOutput, error)
}
