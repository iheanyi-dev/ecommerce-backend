// Package ports defines interfaces required by the Promotion application layer.
package ports

import (
	"context"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/entities"
)

// PromotionRepository defines persistence operations required by promotion use cases.
type PromotionRepository interface {
	// GetByID retrieves a promotion by its unique identifier.
	GetByID(ctx context.Context, promotionID string) (*entities.Promotion, error)

	// Update persists changes to an existing promotion.
	Update(ctx context.Context, promotion *entities.Promotion) error

	// Delete removes an existing promotion by its identifier.
	Delete(ctx context.Context, promotionID string) error

	// CreateIfNoUnexpiredConflict atomically persists a promotion only when
	// no unexpired promotion exists with the same store, product and customer
	// targeting combination.
	//
	// A nil ProductID or CustomerID must be compared as an actual nil value,
	// not treated as a wildcard for uniqueness.
	//
	// A promotion is considered expired when EndsAt <= now, regardless of
	// whether it is scheduled, active, paused or cancelled.
	//
	// Implementations must make the conflict check and insertion atomic
	// to prevent concurrent requests from creating duplicate promotions.
	//
	// Returns apperrors.ErrPromotionConflict when a conflict exists.
	CreateIfNoUnexpiredConflict(
		ctx context.Context,
		promotion *entities.Promotion,
		now time.Time,
	) error

	// ListByStore returns every promotion belonging to a store,
	// regardless of status. Results must be ordered consistently
	// before pagination.
	ListByStore(
		ctx context.Context,
		storeID string,
		limit, offset int,
	) ([]*entities.Promotion, error)

	// ListActiveForTarget returns active promotions belonging to the
	// store that could apply to the supplied product and customer.
	// Implementations must consider all four targeting combinations.
	ListActiveForTarget(
		ctx context.Context,
		storeID, productID, customerID string,
		now time.Time,
	) ([]*entities.Promotion, error)

	// ListAvailableForCustomer returns promotions across all stores
	// that the customer can participate in.
	//
	// Include store-wide promotions and promotions targeting this
	// customer, including negotiated product-specific promotions.
	// Exclude product-only promotions, expired promotions (EndsAt <= now),
	// and cancelled promotions. Scheduled, active, and unexpired paused
	// promotions are eligible. Results must be consistently ordered
	// before pagination.
	ListAvailableForCustomer(
		ctx context.Context,
		customerID string,
		now time.Time,
		limit, offset int,
	) ([]*entities.Promotion, error)
}
