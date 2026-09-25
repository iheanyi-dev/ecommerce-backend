package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
)

// ChangeStorePlanService changes the plan of a Store owned by the
// authenticated user.
type ChangeStorePlanService interface {
	Execute(ctx context.Context, input dto.ChangeStorePlanInput) (dto.StoreOutput, error)
}
