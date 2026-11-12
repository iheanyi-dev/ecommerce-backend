// Package postgres implements Product promotion availability persistence.
package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
	generated "github.com/iheanyi-dev/ecommerce-backend/services/product/infrastructure/persistence/postgres/generated"
)

// ProductPromotionRepository persists the Product promotion availability projection.
type ProductPromotionRepository struct {
	pool    *pgxpool.Pool
	queries *generated.Queries
}

// NewProductPromotionRepository creates the repository using the existing Product pool.
func NewProductPromotionRepository(
	pool *pgxpool.Pool,
	queries *generated.Queries,
) *ProductPromotionRepository {
	return &ProductPromotionRepository{
		pool:    pool,
		queries: queries,
	}
}

var _ ports.ProductPromotionRepository = (*ProductPromotionRepository)(nil)

// Add inserts a promotion availability record.
func (r *ProductPromotionRepository) Add(
	ctx context.Context,
	promotion *entities.ProductPromotion,
) error {
	if promotion == nil {
		return fmt.Errorf("add product promotion: promotion cannot be nil")
	}

	storeID, err := parseUUID(promotion.StoreID())
	if err != nil {
		return fmt.Errorf("add product promotion: invalid store ID: %w", err)
	}

	promotionID, err := parseUUID(promotion.PromotionID())
	if err != nil {
		return fmt.Errorf("add product promotion: invalid promotion ID: %w", err)
	}

	productID, err := nullablePromotionUUID(promotion.ProductID())
	if err != nil {
		return fmt.Errorf("add product promotion: invalid product ID: %w", err)
	}

	err = r.queries.AddProductPromotion(ctx, generated.AddProductPromotionParams{
		StoreID:     toUUID(storeID),
		ProductID:   productID,
		PromotionID: toUUID(promotionID),
		Status:      promotion.Status(),
	})
	if err != nil {
		return translateWriteError("add product promotion", err)
	}

	return nil
}

// Update changes the status of a promotion availability record.
func (r *ProductPromotionRepository) Update(
	ctx context.Context,
	promotion *entities.ProductPromotion,
) error {
	if promotion == nil {
		return fmt.Errorf("update product promotion: promotion cannot be nil")
	}

	storeID, err := parseUUID(promotion.StoreID())
	if err != nil {
		return fmt.Errorf("update product promotion: invalid store ID: %w", err)
	}

	promotionID, err := parseUUID(promotion.PromotionID())
	if err != nil {
		return fmt.Errorf("update product promotion: invalid promotion ID: %w", err)
	}

	err = r.queries.UpdateProductPromotion(ctx, generated.UpdateProductPromotionParams{
		StoreID:     toUUID(storeID),
		PromotionID: toUUID(promotionID),
		Status:      promotion.Status(),
	})
	if err != nil {
		return fmt.Errorf("update product promotion: %w", err)
	}

	return nil
}

// Remove deletes the specified promotion availability record.
func (r *ProductPromotionRepository) Remove(
	ctx context.Context,
	storeID string,
	promotionID string,
) error {
	storeUUID, err := parseUUID(storeID)
	if err != nil {
		return fmt.Errorf("remove product promotion: invalid store ID: %w", err)
	}

	promotionUUID, err := parseUUID(promotionID)
	if err != nil {
		return fmt.Errorf("remove product promotion: invalid promotion ID: %w", err)
	}

	if err := r.queries.RemoveProductPromotion(ctx, generated.RemoveProductPromotionParams{
		StoreID:     toUUID(storeUUID),
		PromotionID: toUUID(promotionUUID),
	}); err != nil {
		return fmt.Errorf("remove product promotion: %w", err)
	}

	return nil
}

// Exists checks for store-wide or matching product-level promotions.
// It deliberately includes upcoming promotions.
func (r *ProductPromotionRepository) Exists(
	ctx context.Context,
	storeID string,
	productID *string,
) (bool, error) {
	storeUUID, err := parseUUID(storeID)
	if err != nil {
		return false, fmt.Errorf("check product promotion: invalid store ID: %w", err)
	}

	productUUID, err := nullablePromotionUUID(productID)
	if err != nil {
		return false, fmt.Errorf("check product promotion: invalid product ID: %w", err)
	}

	exists, err := r.queries.ProductHasPromotion(ctx, generated.ProductHasPromotionParams{
		StoreID:   toUUID(storeUUID),
		ProductID: productUUID,
	})
	if err != nil {
		return false, fmt.Errorf("check product promotion: %w", err)
	}

	return exists, nil
}

func nullablePromotionUUID(value *string) (pgtype.UUID, error) {
	if value == nil {
		return pgtype.UUID{}, nil
	}

	id, err := uuid.Parse(*value)
	if err != nil {
		return pgtype.UUID{}, err
	}

	return toUUID(id), nil
}
