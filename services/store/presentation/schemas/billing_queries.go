package schemas

import (
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
)

type BillingMoneyResponse struct {
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

type BillingPromotionResponse struct {
	Name                string     `json:"name"`
	DiscountAmountMinor int64      `json:"discount_amount_minor"`
	StartsAt            *time.Time `json:"starts_at,omitempty"`
	EndsAt              *time.Time `json:"ends_at,omitempty"`
}

type BillingPlanResponse struct {
	ID              uuid.UUID                 `json:"id"`
	Type            string                    `json:"type"`
	Name            string                    `json:"name"`
	Description     string                    `json:"description"`
	Price           BillingMoneyResponse      `json:"price"`
	DurationSeconds int64                     `json:"duration_seconds"`
	MaxProducts     int                       `json:"max_products"`
	AIEnabled       bool                      `json:"ai_enabled"`
	Available       bool                      `json:"available"`
	Promotion       *BillingPromotionResponse `json:"promotion,omitempty"`
	CreatedAt       *time.Time                `json:"created_at,omitempty"`
	UpdatedAt       *time.Time                `json:"updated_at,omitempty"`
}

type ListBillingPlansResponse struct {
	Plans []BillingPlanResponse `json:"plans"`
}

type BillingSubscriptionResponse struct {
	ID              uuid.UUID            `json:"id"`
	StoreID         uuid.UUID            `json:"store_id"`
	PlanID          uuid.UUID            `json:"plan_id"`
	Price           BillingMoneyResponse `json:"price"`
	DurationSeconds int64                `json:"duration_seconds"`
	Status          string               `json:"status"`
	StartedAt       *time.Time           `json:"started_at,omitempty"`
	ExpiresAt       *time.Time           `json:"expires_at,omitempty"`
	CreatedAt       *time.Time           `json:"created_at,omitempty"`
	UpdatedAt       *time.Time           `json:"updated_at,omitempty"`
}

type ListStoreSubscriptionsResponse struct {
	Subscriptions []BillingSubscriptionResponse `json:"subscriptions"`
}

func NewBillingPlanResponse(output dto.BillingPlanOutput) BillingPlanResponse {
	response := BillingPlanResponse{
		ID:          output.ID,
		Type:        output.Type,
		Name:        output.Name,
		Description: output.Description,
		Price: BillingMoneyResponse{
			AmountMinor: output.Price.AmountMinor,
			Currency:    output.Price.Currency,
		},
		DurationSeconds: int64(output.Duration.Seconds()),
		MaxProducts:     output.MaxProducts,
		AIEnabled:       output.AIEnabled,
		Available:       output.Available,
		CreatedAt:       output.CreatedAt,
		UpdatedAt:       output.UpdatedAt,
	}

	if output.Promotion != nil {
		response.Promotion = &BillingPromotionResponse{
			Name:                output.Promotion.Name,
			DiscountAmountMinor: output.Promotion.DiscountAmountMinor,
			StartsAt:            output.Promotion.StartsAt,
			EndsAt:              output.Promotion.EndsAt,
		}
	}

	return response
}

func NewListBillingPlansResponse(outputs []dto.BillingPlanOutput) ListBillingPlansResponse {
	plans := make([]BillingPlanResponse, 0, len(outputs))
	for _, output := range outputs {
		plans = append(plans, NewBillingPlanResponse(output))
	}
	return ListBillingPlansResponse{Plans: plans}
}

func NewBillingSubscriptionResponse(
	output dto.BillingSubscriptionOutput,
) BillingSubscriptionResponse {
	return BillingSubscriptionResponse{
		ID:      output.ID,
		StoreID: output.StoreID,
		PlanID:  output.PlanID,
		Price: BillingMoneyResponse{
			AmountMinor: output.Price.AmountMinor,
			Currency:    output.Price.Currency,
		},
		DurationSeconds: int64(output.Duration.Seconds()),
		Status:          output.Status,
		StartedAt:       output.StartedAt,
		ExpiresAt:       output.ExpiresAt,
		CreatedAt:       output.CreatedAt,
		UpdatedAt:       output.UpdatedAt,
	}
}

func NewListStoreSubscriptionsResponse(
	outputs []dto.BillingSubscriptionOutput,
) ListStoreSubscriptionsResponse {
	subscriptions := make([]BillingSubscriptionResponse, 0, len(outputs))
	for _, output := range outputs {
		subscriptions = append(subscriptions, NewBillingSubscriptionResponse(output))
	}
	return ListStoreSubscriptionsResponse{Subscriptions: subscriptions}
}
