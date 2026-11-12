package use_cases

import (
	"context"
	"fmt"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
)

type RemoveProductPromotionInput struct {
	StoreID     string
	PromotionID string
}

type RemoveProductPromotion struct {
	repository ports.ProductPromotionRepository
	logger     ports.Logger
	metrics    ports.Metrics
}

func NewRemoveProductPromotion(
	repository ports.ProductPromotionRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *RemoveProductPromotion {
	return &RemoveProductPromotion{
		repository: repository,
		logger:     logger,
		metrics:    metrics,
	}
}

func (uc *RemoveProductPromotion) Execute(
	ctx context.Context,
	input RemoveProductPromotionInput,
) error {
	if uc.repository == nil {
		return fmt.Errorf("product promotion repository is required")
	}

	input.StoreID = strings.TrimSpace(input.StoreID)
	input.PromotionID = strings.TrimSpace(input.PromotionID)

	if input.StoreID == "" {
		recordProductPromotionOperation(ctx, uc.logger, uc.metrics,
			"remove_product_promotion", "product.promotion.remove.validation_failed", "validation")
		return fmt.Errorf("store ID is required")
	}
	if input.PromotionID == "" {
		recordProductPromotionOperation(ctx, uc.logger, uc.metrics,
			"remove_product_promotion", "product.promotion.remove.validation_failed", "validation")
		return fmt.Errorf("promotion ID is required")
	}

	if err := uc.repository.Remove(ctx, input.StoreID, input.PromotionID); err != nil {
		recordProductPromotionOperation(ctx, uc.logger, uc.metrics,
			"remove_product_promotion", "product.promotion.remove.persistence_failed", "persistence")
		return fmt.Errorf("remove product promotion: %w", err)
	}

	recordProductPromotionOperation(ctx, uc.logger, uc.metrics,
		"remove_product_promotion", "product.promotion.remove.succeeded", "")
	return nil
}
