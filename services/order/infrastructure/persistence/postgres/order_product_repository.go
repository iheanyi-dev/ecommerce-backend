package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/infrastructure/persistence/postgres/generated"
)

// OrderProductRepository persists product snapshots belonging to orders.
type OrderProductRepository struct {
	queries *generated.Queries
}

// NewOrderProductRepository creates a repository backed by a PostgreSQL pool.
func NewOrderProductRepository(pool *pgxpool.Pool) *OrderProductRepository {
	return newOrderProductRepository(generated.New(pool))
}

// newOrderProductRepository supports transaction-bound query instances.
func newOrderProductRepository(queries *generated.Queries) *OrderProductRepository {
	return &OrderProductRepository{queries: queries}
}

// CreateMany inserts product snapshots. Use the unit of work when these inserts
// must commit or roll back together with their parent orders.
func (r *OrderProductRepository) CreateMany(
	ctx context.Context,
	products []*entities.OrderProduct,
) error {
	for i, product := range products {
		if product == nil {
			return fmt.Errorf("create order product at index %d: product cannot be nil", i)
		}

		err := r.queries.CreateOrderProduct(ctx, generated.CreateOrderProductParams{
			ID:        toPGUUID(product.ID),
			OrderID:   toPGUUID(product.OrderID),
			ProductID: toPGUUID(product.ProductID),
			Name:      product.Name,
			Qty:       product.Qty,
			Price:     product.Price,
		})
		if err != nil {
			return fmt.Errorf("create order product %s: %w", product.ID, err)
		}
	}

	return nil
}

// FindByOrderIDs retrieves all product snapshots belonging to the supplied orders.
func (r *OrderProductRepository) FindByOrderIDs(
	ctx context.Context,
	orderIDs []uuid.UUID,
) ([]*entities.OrderProduct, error) {
	// Avoid issuing a query with an empty UUID array when there is nothing to find.
	if len(orderIDs) == 0 {
		return []*entities.OrderProduct{}, nil
	}

	ids := make([][16]byte, len(orderIDs))
	for i, id := range orderIDs {
		ids[i] = [16]byte(id)
	}

	pgIDs := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		pgIDs[i] = pgtype.UUID{Bytes: id, Valid: true}
	}

	records, err := r.queries.FindOrderProductsByOrderIDs(ctx, pgIDs)
	if err != nil {
		return nil, fmt.Errorf("find order products: %w", err)
	}

	products := make([]*entities.OrderProduct, 0, len(records))
	for _, record := range records {
		id, err := fromPGUUID(record.ID)
		if err != nil {
			return nil, fmt.Errorf("%w: order product ID: %v", ErrInvalidDatabaseRecord, err)
		}

		orderID, err := fromPGUUID(record.OrderID)
		if err != nil {
			return nil, fmt.Errorf("%w: order product order ID: %v", ErrInvalidDatabaseRecord, err)
		}

		productID, err := fromPGUUID(record.ProductID)
		if err != nil {
			return nil, fmt.Errorf("%w: order product product ID: %v", ErrInvalidDatabaseRecord, err)
		}

		products = append(products, &entities.OrderProduct{
			ID:        id,
			OrderID:   orderID,
			ProductID: productID,
			Name:      record.Name,
			Qty:       record.Qty,
			Price:     record.Price,
		})
	}

	return products, nil
}

// Compile-time check keeps the adapter aligned with the application contract.
var _ ports.OrderProductRepository = (*OrderProductRepository)(nil)
