package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/infrastructure/persistence/postgres/generated"
)

var (
	// ErrOrderNotFound indicates that the requested order does not exist.
	ErrOrderNotFound = errors.New("order not found")

	// ErrInvalidPagination indicates that a repository listing received
	// a negative offset or a non-positive page size.
	ErrInvalidPagination = errors.New("invalid pagination")

	// ErrInvalidDatabaseRecord indicates that persisted data cannot be
	// converted into the corresponding domain entity.
	ErrInvalidDatabaseRecord = errors.New("invalid database record")
)

// OrderRepository implements the order persistence contract using SQLC.
type OrderRepository struct {
	queries *generated.Queries
}

// NewOrderRepository creates an order repository backed by a PostgreSQL pool.
func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return newOrderRepository(generated.New(pool))
}

// newOrderRepository allows the unit of work to supply transaction-bound queries.
func newOrderRepository(queries *generated.Queries) *OrderRepository {
	return &OrderRepository{queries: queries}
}

// CreateMany inserts the supplied orders using the repository's current
// database connection. Call it through OrderUnitOfWork when the inserts
// must be atomic with other writes.
func (r *OrderRepository) CreateMany(
	ctx context.Context,
	orders []*entities.Order,
) error {
	for i, order := range orders {
		if order == nil {
			return fmt.Errorf("create order at index %d: order cannot be nil", i)
		}

		err := r.queries.CreateOrder(ctx, generated.CreateOrderParams{
			ID:               toPGUUID(order.ID),
			CartID:           toPGUUID(order.CartID),
			StoreID:          toPGUUID(order.StoreID),
			CustomerID:       toPGUUID(order.CustomerID),
			DeliveryLocation: order.DeliveryLocation,
			PriceProduct:     order.PriceProduct,
			PriceShipping:    order.PriceShipping,
			Currency:         order.Currency,
			State:            string(order.State),
			CreatedAt:        pgtype.Timestamptz{Time: order.CreatedAt, Valid: true},
			UpdatedAt:        pgtype.Timestamptz{Time: order.UpdatedAt, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("create order %s: %w", order.ID, err)
		}
	}

	return nil
}

// FindByID retrieves an order by its identifier.
func (r *OrderRepository) FindByID(
	ctx context.Context,
	orderID uuid.UUID,
) (*entities.Order, error) {
	record, err := r.queries.GetOrderByID(ctx, toPGUUID(orderID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find order %s: %w", orderID, err)
	}

	order, err := mapOrder(record)
	if err != nil {
		return nil, fmt.Errorf("map order %s: %w", orderID, err)
	}

	return order, nil
}

// Update persists the mutable order fields supported by the current SQL query.
func (r *OrderRepository) Update(
	ctx context.Context,
	order *entities.Order,
) error {
	if order == nil {
		return errors.New("update order: order cannot be nil")
	}

	rowsAffected, err := r.queries.UpdateOrder(ctx, generated.UpdateOrderParams{
		ID:            toPGUUID(order.ID),
		PriceShipping: order.PriceShipping,
		State:         string(order.State),
		UpdatedAt:     pgtype.Timestamptz{Time: order.UpdatedAt, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("update order %s: %w", order.ID, err)
	}
	if rowsAffected == 0 {
		return ErrOrderNotFound
	}

	return nil
}

// ListByCustomerID returns a page of orders and the customer's total order count.
func (r *OrderRepository) ListByCustomerID(
	ctx context.Context,
	customerID uuid.UUID,
	offset, limit int,
) ([]*entities.Order, int64, error) {
	if offset < 0 || limit <= 0 {
		return nil, 0, ErrInvalidPagination
	}

	records, err := r.queries.ListOrdersByCustomerID(
		ctx,
		generated.ListOrdersByCustomerIDParams{
			CustomerID: toPGUUID(customerID),
			PageOffset: int64(offset),
			PageLimit:  int64(limit),
		},
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list orders for customer %s: %w", customerID, err)
	}

	total, err := r.queries.CountOrdersByCustomerID(ctx, toPGUUID(customerID))
	if err != nil {
		return nil, 0, fmt.Errorf("count orders for customer %s: %w", customerID, err)
	}

	orders, err := mapOrders(records)
	if err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// ListByStoreID returns a page of orders and the store's total order count.
func (r *OrderRepository) ListByStoreID(
	ctx context.Context,
	storeID uuid.UUID,
	offset, limit int,
) ([]*entities.Order, int64, error) {
	if offset < 0 || limit <= 0 {
		return nil, 0, ErrInvalidPagination
	}

	records, err := r.queries.ListOrdersByStoreID(
		ctx,
		generated.ListOrdersByStoreIDParams{
			StoreID:    toPGUUID(storeID),
			PageOffset: int64(offset),
			PageLimit:  int64(limit),
		},
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list orders for store %s: %w", storeID, err)
	}

	total, err := r.queries.CountOrdersByStoreID(ctx, toPGUUID(storeID))
	if err != nil {
		return nil, 0, fmt.Errorf("count orders for store %s: %w", storeID, err)
	}

	orders, err := mapOrders(records)
	if err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// mapOrders converts generated database records into domain entities.
func mapOrders(records []generated.Order) ([]*entities.Order, error) {
	orders := make([]*entities.Order, 0, len(records))

	for _, record := range records {
		order, err := mapOrder(record)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// mapOrder converts one generated database record into a domain entity.
func mapOrder(record generated.Order) (*entities.Order, error) {
	id, err := fromPGUUID(record.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: order ID: %v", ErrInvalidDatabaseRecord, err)
	}

	cartID, err := fromPGUUID(record.CartID)
	if err != nil {
		return nil, fmt.Errorf("%w: order cart ID: %v", ErrInvalidDatabaseRecord, err)
	}

	storeID, err := fromPGUUID(record.StoreID)
	if err != nil {
		return nil, fmt.Errorf("%w: order store ID: %v", ErrInvalidDatabaseRecord, err)
	}

	customerID, err := fromPGUUID(record.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("%w: order customer ID: %v", ErrInvalidDatabaseRecord, err)
	}

	if !record.CreatedAt.Valid || !record.UpdatedAt.Valid {
		return nil, fmt.Errorf("%w: order timestamps are invalid", ErrInvalidDatabaseRecord)
	}

	return &entities.Order{
		ID:               id,
		CartID:           cartID,
		StoreID:          storeID,
		CustomerID:       customerID,
		DeliveryLocation: record.DeliveryLocation,
		PriceProduct:     record.PriceProduct,
		PriceShipping:    record.PriceShipping,
		Currency:         record.Currency,
		State:            entities.OrderState(record.State),
		CreatedAt:        record.CreatedAt.Time,
		UpdatedAt:        record.UpdatedAt.Time,
	}, nil
}

// toPGUUID converts a domain UUID into the representation expected by pgx.
func toPGUUID(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(value), Valid: true}
}

// fromPGUUID converts a database UUID into the domain UUID representation.
func fromPGUUID(value pgtype.UUID) (uuid.UUID, error) {
	if !value.Valid {
		return uuid.Nil, errors.New("UUID is NULL")
	}

	return uuid.UUID(value.Bytes), nil
}

// Compile-time check keeps the adapter aligned with the application contract.
var _ ports.OrderRepository = (*OrderRepository)(nil)
