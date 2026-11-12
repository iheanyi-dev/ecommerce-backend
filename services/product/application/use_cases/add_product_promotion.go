package use_cases

import (
	"context"
	"fmt"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
)

type AddProductPromotionInput struct {
	StoreID     string
	ProductID   *string
	PromotionID string
	Status      string
}

type AddProductPromotion struct {
	repository ports.ProductPromotionRepository
	logger     ports.Logger
	metrics    ports.Metrics
}

func NewAddProductPromotion(
	repository ports.ProductPromotionRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *AddProductPromotion {
	return &AddProductPromotion{
		repository: repository,
		logger:     logger,
		metrics:    metrics,
	}
}

func (uc *AddProductPromotion) Execute(
	ctx context.Context,
	input AddProductPromotionInput,
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
			"add_product_promotion", "product.promotion.add.validation_failed", "validation")
		return fmt.Errorf("create product promotion: %w", err)
	}

	if err := uc.repository.Add(ctx, promotion); err != nil {
		recordProductPromotionOperation(ctx, uc.logger, uc.metrics,
			"add_product_promotion", "product.promotion.add.persistence_failed", "persistence")
		return fmt.Errorf("add product promotion: %w", err)
	}

	recordProductPromotionOperation(ctx, uc.logger, uc.metrics,
		"add_product_promotion", "product.promotion.add.succeeded", "")
	return nil
}
