package use_cases

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/domain/entities"
)

// InitiatePayoutInput identifies the received order and supplies its beneficiary.
type InitiatePayoutInput struct {
	OrderID     uuid.UUID
	StoreID     uuid.UUID
	Beneficiary ports.PayoutBeneficiary
}

// InitiatePayoutOutput contains the payout operation returned by Payment.
type InitiatePayoutOutput struct {
	PayoutID      string
	AttemptID     string
	Reference     string
	Status        string
	AttemptStatus string
	Provider      string
	AmountMinor   int64
	Currency      string
}

// InitiatePayout starts settlement for an order whose receipt is confirmed.
type InitiatePayout struct {
	orders  ports.OrderRepository
	payouts ports.PayoutClient
	logger  ports.Logger
	metrics ports.Metrics
}

// NewInitiatePayout constructs the InitiatePayout use case.
func NewInitiatePayout(
	orders ports.OrderRepository,
	payouts ports.PayoutClient,
	logger ports.Logger,
	metrics ports.Metrics,
) *InitiatePayout {
	return &InitiatePayout{
		orders:  orders,
		payouts: payouts,
		logger:  logger,
		metrics: metrics,
	}
}

// Execute pays the store 95% of the product subtotal plus the full shipping
// amount. The 5% platform commission is rounded down to the nearest minor unit.
// The order state is changed only after a trusted successful payout event.
func (uc *InitiatePayout) Execute(
	ctx context.Context,
	input InitiatePayoutInput,
) (*InitiatePayoutOutput, error) {
	const operation = "initiate_payout"

	if input.OrderID == uuid.Nil {
		return nil, fmt.Errorf("%s: order ID is required", operation)
	}
	if input.StoreID == uuid.Nil {
		return nil, fmt.Errorf("%s: store ID is required", operation)
	}
	if err := validatePayoutBeneficiary(input.Beneficiary); err != nil {
		uc.log(ctx, operation, "invalid_beneficiary")
		return nil, fmt.Errorf("%s: %w", operation, err)
	}

	order, err := uc.orders.FindByID(ctx, input.OrderID)
	if err != nil {
		uc.log(ctx, operation, "load_order")
		return nil, fmt.Errorf("%s: load order: %w", operation, err)
	}
	if order == nil || order.ID != input.OrderID {
		uc.log(ctx, operation, "load_order")
		return nil, fmt.Errorf("%s: order not found", operation)
	}
	if order.StoreID != input.StoreID {
		uc.log(ctx, operation, "store_mismatch")
		return nil, fmt.Errorf("%s: order does not belong to the supplied store", operation)
	}
	if order.State != entities.OrderStateReceived {
		uc.log(ctx, operation, "invalid_order_state")
		return nil, fmt.Errorf("%s: order state %q does not allow payout initiation",
			operation, order.State)
	}

	// Validate both components and their combined value before calculating
	// the payout. TotalPrice also detects negative values and int64 overflow.
	if _, err := order.TotalPrice(); err != nil {
		uc.log(ctx, operation, "calculate_total")
		return nil, fmt.Errorf("%s: validate order prices: %w", operation, err)
	}

	commission := order.PriceProduct / 20
	payoutAmount := order.PriceProduct - commission + order.PriceShipping
	if payoutAmount <= 0 {
		uc.log(ctx, operation, "invalid_amount")
		return nil, fmt.Errorf("%s: payout amount must be positive", operation)
	}

	response, err := uc.payouts.InitiatePayout(ctx, ports.InitiatePayoutRequest{
		BusinessReferenceType: "order",
		BusinessReferenceID:   order.ID,
		StoreID:               order.StoreID,
		OrderID:               order.ID,
		Reason:                "order_settlement",
		AmountMinor:           payoutAmount,
		Currency:              order.Currency,
		Beneficiary:           input.Beneficiary,
	})
	if err != nil {
		uc.log(ctx, operation, "payment_service")
		return nil, fmt.Errorf("%s: initiate payout through Payment service: %w", operation, err)
	}
	if response == nil {
		uc.log(ctx, operation, "invalid_payout_response")
		return nil, fmt.Errorf("%s: Payment service returned a nil payout response", operation)
	}

	uc.log(ctx, operation, "")
	_ = uc.metrics.Increment(ctx, ports.Metric{
		Name:   "order_payouts_initiated",
		Value:  1,
		Labels: map[string]string{"operation": operation},
	})

	return &InitiatePayoutOutput{
		PayoutID:      response.PayoutID,
		AttemptID:     response.AttemptID,
		Reference:     response.Reference,
		Status:        response.Status,
		AttemptStatus: response.AttemptStatus,
		Provider:      response.Provider,
		AmountMinor:   response.AmountMinor,
		Currency:      response.Currency,
	}, nil
}

func validatePayoutBeneficiary(beneficiary ports.PayoutBeneficiary) error {
	fields := map[string]string{
		"account name":   beneficiary.AccountName,
		"account number": beneficiary.AccountNumber,
		"bank code":      beneficiary.BankCode,
		"bank name":      beneficiary.BankName,
		"country code":   beneficiary.CountryCode,
	}
	for name, value := range fields {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("beneficiary %s is required", name)
		}
	}
	return nil
}

// log records outcomes without allowing observability failures to affect payout.
func (uc *InitiatePayout) log(ctx context.Context, operation, failureCategory string) {
	_ = uc.logger.Log(ctx, ports.LogEvent{
		Event:           "order.payout",
		Operation:       operation,
		FailureCategory: failureCategory,
	})
}
