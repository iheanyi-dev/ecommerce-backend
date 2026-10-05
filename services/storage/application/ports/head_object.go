package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/dto"
)

// HeadObjectService checks whether an object exists and returns its metadata.
//
// Storage performs no business-level authorization. The trusted internal
// caller owns the authorization decision before invoking this operation.
type HeadObjectService interface {
	Execute(
		ctx context.Context,
		input dto.HeadObjectInput,
	) (dto.HeadObjectOutput, error)
}
