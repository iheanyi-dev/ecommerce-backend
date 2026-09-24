package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/entities"
)

// StoreRepository defines persistence operations required by the Store
// application layer.
//
// The application depends only on this capability and does not know which
// persistence technology provides the implementation.
type StoreRepository interface {
	// Create persists a newly created Store aggregate.
	Create(ctx context.Context, store *entities.Store) error

	// Delete removes a Store aggregate by its identifier.
	//
	// The application layer uses this operation as a compensating action
	// when Store persistence has succeeded but the subsequent synchronous
	// Identity Service update fails.
	Delete(ctx context.Context, storeID uuid.UUID) error

	// FindByID retrieves a Store by its identifier.
	FindByID(ctx context.Context, storeID uuid.UUID) (*entities.Store, error)

	// FindByOwnerID retrieves the Store owned by the supplied user.
	FindByOwnerID(ctx context.Context, ownerID uuid.UUID) (*entities.Store, error)

	// FindBySlug retrieves a Store by its slug.
	FindBySlug(ctx context.Context, slug string) (*entities.Store, error)
	// ExistsBySlug reports whether a Store already uses the supplied slug.
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
	// ListActive retrieves all Stores that are currently active.
	ListActive(
		ctx context.Context,
		query string,
		page int,
		pageSize int,
	) ([]*entities.Store, int, error)

	// Update persists changes made to an existing Store aggregate.
	Update(ctx context.Context, store *entities.Store) error

	// ChangePlan persists a Store aggregate after its plan has been changed.
	//
	// The complete aggregate is supplied because changing a plan may also
	// change other aggregate state as part of the Store's plan-transition
	// workflow, such as its status.
	ChangePlan(ctx context.Context, store *entities.Store) error
	// ChangeStatus persists a Store status transition independently of the
	// Store plan workflow.
	//
	// The Store aggregate owns the status transition itself. The repository
	// receives only the resulting Store identifier and validated status
	// because Billing/Subscription status changes are independent of plan
	// persistence.
	ChangeStatus(
		ctx context.Context,
		storeID uuid.UUID,
		status string,
	) error
}
