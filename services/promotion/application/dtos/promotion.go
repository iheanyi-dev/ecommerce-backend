package dtos

import (
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/entities"
)

// PromotionDTO represents a promotion returned by an application use case.
type PromotionDTO struct {
	ID                    string
	StoreID               string
	ProductID             *string
	CustomerID            *string
	PercentageBasisPoints *int64
	FixedAmount           *MoneyDTO
	StartsAt              time.Time
	EndsAt                time.Time
	Status                entities.PromotionStatus
	CreatedAt             time.Time
	UpdatedAt             time.Time
}
