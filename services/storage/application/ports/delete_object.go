package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/dto"
)

// DeleteObjectService deletes a generic object.
//
// The calling service owns the object's business meaning and authorization.
type DeleteObjectService interface {
	Execute(
		ctx context.Context,
		input dto.DeleteObjectInput,
	) error
}
