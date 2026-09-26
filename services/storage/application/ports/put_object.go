package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/dto"
)

// PutObjectService stores a generic object.
//
// The application contract accepts an io.Reader so transport adapters can
// stream object content without loading the complete object into memory.
type PutObjectService interface {
	Execute(
		ctx context.Context,
		input dto.PutObjectInput,
	) error
}
