package schemas

import (
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/dtos"
)

type MoneyResponse struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type PromotionResponse struct {
	ID                    string         `json:"id"`
	StoreID               string         `json:"store_id"`
	ProductID             *string        `json:"product_id,omitempty"`
	CustomerID            *string        `json:"customer_id,omitempty"`
	PercentageBasisPoints *int64         `json:"percentage_basis_points,omitempty"`
	FixedAmount           *MoneyResponse `json:"fixed_amount,omitempty"`
	StartsAt              time.Time      `json:"starts_at"`
	EndsAt                time.Time      `json:"ends_at"`
	Status                string         `json:"status"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
}

func NewPromotionResponse(dto dtos.PromotionDTO) PromotionResponse {
	var fixedAmount *MoneyResponse

	if dto.FixedAmount != nil {
		fixedAmount = &MoneyResponse{
			Amount:   dto.FixedAmount.Amount,
			Currency: dto.FixedAmount.Currency,
		}
	}

	return PromotionResponse{
		ID:                    dto.ID,
		StoreID:               dto.StoreID,
		ProductID:             dto.ProductID,
		CustomerID:            dto.CustomerID,
		PercentageBasisPoints: dto.PercentageBasisPoints,
		FixedAmount:           fixedAmount,
		StartsAt:              dto.StartsAt,
		EndsAt:                dto.EndsAt,
		Status:                string(dto.Status),
		CreatedAt:             dto.CreatedAt,
		UpdatedAt:             dto.UpdatedAt,
	}
}

func NewPromotionResponses(dtos []dtos.PromotionDTO) []PromotionResponse {
	result := make([]PromotionResponse, 0, len(dtos))
	for _, dto := range dtos {
		result = append(result, NewPromotionResponse(dto))
	}
	return result
}
