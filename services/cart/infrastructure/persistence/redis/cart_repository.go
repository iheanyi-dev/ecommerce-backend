package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/entities"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/errors"
)

// CartRepository persists carts in Redis.
//
// Each customer's cart is stored as one JSON document. Keeping the cart in a
// single key makes each save atomic and preserves the ordering of its items.
type CartRepository struct {
	client *goredis.Client
}

type cartDocument struct {
	ID         uuid.UUID          `json:"id"`
	CustomerID uuid.UUID          `json:"customer_id"`
	Items      []cartItemDocument `json:"items"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

type cartItemDocument struct {
	ID        uuid.UUID `json:"id"`
	ProductID uuid.UUID `json:"product_id"`
	StoreID   uuid.UUID `json:"store_id"`
	Quantity  int64     `json:"quantity"`
}

func NewCartRepository(client *goredis.Client) *CartRepository {
	return &CartRepository{client: client}
}

func (r *CartRepository) GetByCustomerID(
	ctx context.Context,
	customerID uuid.UUID,
) (*entities.Cart, error) {
	data, err := r.client.Get(ctx, cartKey(customerID)).Bytes()
	if err != nil {
		if err == goredis.Nil {
			return nil, domainerrors.ErrCartNotFound
		}
		return nil, fmt.Errorf("get cart from Redis: %w", err)
	}

	var doc cartDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("decode cart from Redis: %w", err)
	}

	items := make([]entities.CartItemState, 0, len(doc.Items))
	for _, item := range doc.Items {
		items = append(items, entities.CartItemState{
			ID:        item.ID,
			ProductID: item.ProductID,
			StoreID:   item.StoreID,
			Quantity:  item.Quantity,
		})
	}

	cart, err := entities.ReconstituteCart(
		doc.ID,
		doc.CustomerID,
		items,
		doc.CreatedAt,
		doc.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("reconstitute cart from Redis: %w", err)
	}

	return cart, nil
}

func (r *CartRepository) Save(
	ctx context.Context,
	cart *entities.Cart,
) error {
	if cart == nil {
		return fmt.Errorf("save cart to Redis: cart is nil")
	}

	itemEntities := cart.Items()
	doc := cartDocument{
		ID:         cart.ID(),
		CustomerID: cart.CustomerID(),
		Items:      make([]cartItemDocument, 0, len(itemEntities)),
		CreatedAt:  cart.CreatedAt(),
		UpdatedAt:  cart.UpdatedAt(),
	}

	for _, item := range itemEntities {
		doc.Items = append(doc.Items, cartItemDocument{
			ID:        item.ID(),
			ProductID: item.ProductID(),
			StoreID:   item.StoreID(),
			Quantity:  item.Quantity(),
		})
	}

	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("encode cart for Redis: %w", err)
	}

	if err := r.client.Set(ctx, cartKey(cart.CustomerID()), data, 0).Err(); err != nil {
		return fmt.Errorf("save cart to Redis: %w", err)
	}

	return nil
}

func (r *CartRepository) DeleteByCustomerID(
	ctx context.Context,
	customerID uuid.UUID,
) error {
	if err := r.client.Del(ctx, cartKey(customerID)).Err(); err != nil {
		return fmt.Errorf("delete cart from Redis: %w", err)
	}
	return nil
}

func cartKey(customerID uuid.UUID) string {
	return "cart:" + customerID.String()
}
