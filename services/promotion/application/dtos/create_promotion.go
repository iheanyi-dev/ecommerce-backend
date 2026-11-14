package dtos

import "time"

// CreatePromotionInput contains the data required to create a promotion.
//
// A nil ProductID targets every product in the store.
// A nil CustomerID makes the promotion available to all customers.
type CreatePromotionInput struct {
	StoreID               string
	ProductID             *string
	CustomerID            *string
	PercentageBasisPoints *int64
	FixedAmount           *MoneyDTO
	StartsAt              time.Time
	EndsAt                time.Time
}
