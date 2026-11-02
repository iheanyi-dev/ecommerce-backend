package schemas

import (
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
)

type InitiatePaymentRequest struct {
	PayerID         string `json:"payer_id"`
	Purpose         string `json:"purpose"`
	BusinessRefType string `json:"business_reference_type"`
	BusinessRefID   string `json:"business_reference_id"`
	Amount          int64  `json:"amount"`
	Currency        string `json:"currency"`
	CustomerEmail   string `json:"customer_email"`
}

type InitiatePaymentResponse struct {
	PaymentID     string `json:"payment_id"`
	AttemptID     string `json:"attempt_id"`
	PaymentStatus string `json:"payment_status"`
	AttemptStatus string `json:"attempt_status"`
	Provider      string `json:"provider"`
	CheckoutURL   string `json:"checkout_url"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
}

type PaymentAttemptResponse struct {
	ID                string      `json:"id"`
	PaymentID         string      `json:"payment_id"`
	Provider          string      `json:"provider"`
	ProviderReference string      `json:"provider_reference,omitempty"`
	Status            string      `json:"status"`
	FailureReason     string      `json:"failure_reason,omitempty"`
	CheckoutURL       string      `json:"checkout_url,omitempty"`
	ExpiresAt         interface{} `json:"expires_at,omitempty"`
}

type PaymentResponse struct {
	ID                    string                  `json:"id"`
	Reference             string                  `json:"reference"`
	PayerID               string                  `json:"payer_id"`
	Purpose               string                  `json:"purpose"`
	BusinessReferenceType string                  `json:"business_reference_type"`
	BusinessReferenceID   string                  `json:"business_reference_id"`
	Amount                int64                   `json:"amount"`
	Currency              string                  `json:"currency"`
	Status                string                  `json:"status"`
	Attempt               *PaymentAttemptResponse `json:"attempt,omitempty"`
}

type PaymentListResponse struct {
	Items      []PaymentResponse `json:"items"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	Total      int64             `json:"total"`
	TotalPages int               `json:"total_pages"`
}

type InitiatePayoutRequest struct {
	BusinessReferenceType string                     `json:"business_reference_type"`
	BusinessReferenceID   string                     `json:"business_reference_id"`
	StoreID               string                     `json:"store_id"`
	OrderID               string                     `json:"order_id"`
	Reason                string                     `json:"reason"`
	Amount                int64                      `json:"amount"`
	Currency              string                     `json:"currency"`
	Beneficiary           entities.PayoutBeneficiary `json:"beneficiary"`
}

type PayoutResponse struct {
	ID                    string    `json:"id"`
	Reference             string    `json:"reference"`
	BusinessReferenceType string    `json:"business_reference_type,omitempty"`
	BusinessReferenceID   string    `json:"business_reference_id,omitempty"`
	StoreID               string    `json:"store_id"`
	OrderID               string    `json:"order_id"`
	Reason                string    `json:"reason"`
	Amount                int64     `json:"amount"`
	Currency              string    `json:"currency"`
	Status                string    `json:"status"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type PayoutListResponse struct {
	Items  []PayoutResponse `json:"items"`
	Total  int64            `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

type PayoutOperationResponse struct {
	PayoutID      string `json:"payout_id"`
	AttemptID     string `json:"attempt_id"`
	Reference     string `json:"reference"`
	Status        string `json:"status"`
	AttemptStatus string `json:"attempt_status"`
	Provider      string `json:"provider"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
}

func MapPaymentDetails(details *use_cases.PaymentDetails) PaymentResponse {
	response := PaymentResponse{
		ID:                    details.ID,
		Reference:             details.Reference,
		PayerID:               details.PayerID,
		Purpose:               string(details.Purpose),
		BusinessReferenceType: details.BusinessReferenceType,
		BusinessReferenceID:   details.BusinessReferenceID,
		Amount:                details.Amount,
		Currency:              details.Currency,
		Status:                string(details.Status),
	}

	if details.Attempt != nil {
		response.Attempt = &PaymentAttemptResponse{
			ID:                details.Attempt.ID,
			PaymentID:         details.Attempt.PaymentID,
			Provider:          details.Attempt.Provider,
			ProviderReference: details.Attempt.ProviderReference,
			Status:            string(details.Attempt.Status),
			FailureReason:     details.Attempt.FailureReason,
			CheckoutURL:       details.Attempt.CheckoutURL,
			ExpiresAt:         details.Attempt.ExpiresAt,
		}
	}

	return response
}

func MapPayout(payout *entities.Payout) PayoutResponse {
	reference := payout.BusinessReference()

	return PayoutResponse{
		ID:                    payout.ID(),
		Reference:             payout.Reference(),
		BusinessReferenceType: reference.Type(),
		BusinessReferenceID:   reference.ID(),
		StoreID:               payout.StoreID(),
		OrderID:               payout.OrderID(),
		Reason:                payout.Reason(),
		Amount:                payout.Amount().Amount(),
		Currency:              payout.Amount().Currency(),
		Status:                string(payout.Status()),
		CreatedAt:             payout.CreatedAt(),
		UpdatedAt:             payout.UpdatedAt(),
	}
}

func MapPayoutOperation(output *use_cases.InitiatePayoutOutput) PayoutOperationResponse {
	return PayoutOperationResponse{
		PayoutID:      output.PayoutID,
		AttemptID:     output.AttemptID,
		Reference:     output.Reference,
		Status:        string(output.Status),
		AttemptStatus: string(output.AttemptStatus),
		Provider:      output.Provider,
		Amount:        output.Amount,
		Currency:      output.Currency,
	}
}
