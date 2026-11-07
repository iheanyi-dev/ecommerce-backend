package dto

import "github.com/google/uuid"

// FindAssociatedProductsInput contains the Product and page for behavioral association search.
type FindAssociatedProductsInput struct {
	ProductID uuid.UUID
	Page      int
}

// FindAssociatedProductsOutput contains the requested page of associated Products.
type FindAssociatedProductsOutput struct {
	Recommendations []ProductRecommendation
}
