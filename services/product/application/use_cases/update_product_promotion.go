package use_cases

import (
	"context"
	"fmt"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
)

type UpdateProductPromotionInput struct {
	StoreID     string
	ProductID   *string
	PromotionID string
	Status      string
}

type UpdateProductPromotion struct {
	repository ports.ProductPromotionRepository
	logger     ports.Logger
	metrics    ports.Metrics
}

func NewUpdateProductPromotion(
	repository ports.ProductPromotionRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *UpdateProductPromotion {
	return &UpdateProductPromotion{
		repository: repository,
		logger:     logger,
		metrics:    metrics,
	}
}

func (uc *UpdateProductPromotion) Execute(
	ctx context.Context,
	input UpdateProductPromotionInput,
) error {
	if uc.repository == nil {
		return fmt.Errorf("product promotion repository is required")
	}

	promotion, err := entities.NewProductPromotion(
		input.StoreID,
		input.ProductID,
		input.PromotionID,
		input.Status,
	)
	if err != nil {
		recordProductPromotionOperation(ctx, uc.logger, uc.metrics,
			"update_product_promotion", "product.promotion.update.validation_failed", "validation")
		return fmt.Errorf("create product promotion update: %w", err)
	}

	if err := uc.repository.Update(ctx, promotion); err != nil {
		recordProductPromotionOperation(ctx, uc.logger, uc.metrics,
			"update_product_promotion", "product.promotion.update.persistence_failed", "persistence")
		return fmt.Errorf("update product promotion: %w", err)
	}

	recordProductPromotionOperation(ctx, uc.logger, uc.metrics,
		"update_product_promotion", "product.promotion.update.succeeded", "")
	return nil
}
