package entities

import (
	"fmt"
	"strings"
)

// ProductPromotion represents promotion availability for a store
// or an individual product.
type ProductPromotion struct {
	storeID     string
	productID   *string
	promotionID string
	status      string
}

// NewProductPromotion creates a promotion availability record.
func NewProductPromotion(
	storeID string,
	productID *string,
	promotionID string,
	status string,
) (*ProductPromotion, error) {
	storeID = strings.TrimSpace(storeID)
	promotionID = strings.TrimSpace(promotionID)
	status = strings.TrimSpace(status)

	if storeID == "" {
		return nil, fmt.Errorf("store ID is required")
	}
	if promotionID == "" {
		return nil, fmt.Errorf("promotion ID is required")
	}
	if status == "" {
		return nil, fmt.Errorf("promotion status is required")
	}

	var normalizedProductID *string
	if productID != nil {
		id := strings.TrimSpace(*productID)
		if id == "" {
			return nil, fmt.Errorf("product ID cannot be blank")
		}
		normalizedProductID = &id
	}

	return &ProductPromotion{
		storeID:     storeID,
		productID:   normalizedProductID,
		promotionID: promotionID,
		status:      status,
	}, nil
}

func (p *ProductPromotion) StoreID() string {
	return p.storeID
}

func (p *ProductPromotion) ProductID() *string {
	if p.productID == nil {
		return nil
	}
	id := *p.productID
	return &id
}

func (p *ProductPromotion) PromotionID() string {
	return p.promotionID
}

func (p *ProductPromotion) Status() string {
	return p.status
}
