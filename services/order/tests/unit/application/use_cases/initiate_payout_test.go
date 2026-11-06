package use_cases_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/domain/entities"
)

type initiatePayoutClientMock struct {
	request  ports.InitiatePayoutRequest
	response *ports.InitiatePayoutResponse
	err      error
	calls    int
}

func (m *initiatePayoutClientMock) InitiatePayout(
	_ context.Context,
	request ports.InitiatePayoutRequest,
) (*ports.InitiatePayoutResponse, error) {
	m.calls++
	m.request = request
	return m.response, m.err
}

func validPayoutBeneficiary() ports.PayoutBeneficiary {
	return ports.PayoutBeneficiary{
		AccountName:   "Example Store",
		AccountNumber: "0123456789",
		BankCode:      "058",
		BankName:      "Example Bank",
		CountryCode:   "NG",
	}
}

func newReceivedOrder(t *testing.T) *entities.Order {
	t.Helper()

	order := newInitiatePaymentOrder(t, entities.OrderStateShippingAdded)
	if err := order.MarkPaymentSuccessful(); err != nil {
		t.Fatalf("MarkPaymentSuccessful() error = %v", err)
	}
	if err := order.MarkSent(order.StoreID); err != nil {
		t.Fatalf("MarkSent() error = %v", err)
	}
	if err := order.ConfirmReceipt(order.CustomerID); err != nil {
		t.Fatalf("ConfirmReceipt() error = %v", err)
	}
	return order
}

func TestInitiatePayoutCalculatesStoreSettlementAndReturnsResponse(t *testing.T) {
	order := newReceivedOrder(t)
	repo := &initiatePaymentOrderRepository{order: order}
	payouts := &initiatePayoutClientMock{
		response: &ports.InitiatePayoutResponse{
			PayoutID:      "payout-id",
			AttemptID:     "attempt-id",
			Reference:     "provider-reference",
			Status:        "pending",
			AttemptStatus: "processing",
			Provider:      "paystack",
			AmountMinor:   14375,
			Currency:      "NGN",
		},
	}

	uc := use_cases.NewInitiatePayout(
		repo, payouts, initiatePaymentLoggerMock{}, initiatePaymentMetricsMock{},
	)
	result, err := uc.Execute(context.Background(), use_cases.InitiatePayoutInput{
		OrderID:     order.ID,
		StoreID:     order.StoreID,
		Beneficiary: validPayoutBeneficiary(),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.PayoutID != "payout-id" ||
		result.AttemptID != "attempt-id" ||
		result.Reference != "provider-reference" ||
		result.Status != "pending" ||
		result.AttemptStatus != "processing" ||
		result.Provider != "paystack" ||
		result.AmountMinor != 14375 ||
		result.Currency != "NGN" {
		t.Fatalf("unexpected payout result: %+v", result)
	}

	if payouts.calls != 1 {
		t.Fatalf("Payment payout calls = %d, want 1", payouts.calls)
	}

	request := payouts.request
	if request.BusinessReferenceType != "order" ||
		request.BusinessReferenceID != order.ID ||
		request.StoreID != order.StoreID ||
		request.OrderID != order.ID ||
		request.Reason != "order_settlement" ||
		request.AmountMinor != 14375 ||
		request.Currency != order.Currency ||
		request.Beneficiary != validPayoutBeneficiary() {
		t.Fatalf("unexpected Payment payout request: %+v", request)
	}

	if repo.calls != 1 {
		t.Fatalf("FindByID calls = %d, want 1", repo.calls)
	}
	if order.State != entities.OrderStateReceived {
		t.Fatalf("order state changed to %q, want %q", order.State, entities.OrderStateReceived)
	}
}

func TestInitiatePayoutRoundsCommissionDown(t *testing.T) {
	order := newReceivedOrder(t)
	order.PriceProduct = 10001
	order.PriceShipping = 999

	payouts := &initiatePayoutClientMock{
		response: &ports.InitiatePayoutResponse{PayoutID: "payout-id"},
	}
	uc := use_cases.NewInitiatePayout(
		&initiatePaymentOrderRepository{order: order},
		payouts,
		initiatePaymentLoggerMock{},
		initiatePaymentMetricsMock{},
	)

	_, err := uc.Execute(context.Background(), use_cases.InitiatePayoutInput{
		OrderID:     order.ID,
		StoreID:     order.StoreID,
		Beneficiary: validPayoutBeneficiary(),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// 5% of 10001 is 500.05 minor units; commission is rounded down to 500.
	if payouts.request.AmountMinor != 10500 {
		t.Fatalf("payout amount = %d, want 10500", payouts.request.AmountMinor)
	}
}

func TestInitiatePayoutRejectsInvalidInputAndOrderState(t *testing.T) {
	tests := []struct {
		name        string
		orderID     uuid.UUID
		state       entities.OrderState
		beneficiary ports.PayoutBeneficiary
	}{
		{
			name:        "missing order ID",
			state:       entities.OrderStateReceived,
			beneficiary: validPayoutBeneficiary(),
		},
		{
			name:        "missing account name",
			orderID:     uuid.New(),
			state:       entities.OrderStateReceived,
			beneficiary: ports.PayoutBeneficiary{AccountNumber: "0123456789", BankCode: "058", BankName: "Bank", CountryCode: "NG"},
		},
		{
			name:        "order not received",
			orderID:     uuid.New(),
			state:       entities.OrderStateSent,
			beneficiary: validPayoutBeneficiary(),
		},
		{
			name:        "order already completed",
			orderID:     uuid.New(),
			state:       entities.OrderStateTransactionCompleted,
			beneficiary: validPayoutBeneficiary(),
		},
		{
			name:        "order payment failed",
			orderID:     uuid.New(),
			state:       entities.OrderStatePaymentFailed,
			beneficiary: validPayoutBeneficiary(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := newReceivedOrder(t)
			order.ID = tt.orderID
			order.State = tt.state

			payouts := &initiatePayoutClientMock{
				response: &ports.InitiatePayoutResponse{PayoutID: "payout-id"},
			}
			uc := use_cases.NewInitiatePayout(
				&initiatePaymentOrderRepository{order: order},
				payouts,
				initiatePaymentLoggerMock{},
				initiatePaymentMetricsMock{},
			)

			_, err := uc.Execute(context.Background(), use_cases.InitiatePayoutInput{
				OrderID:     tt.orderID,
				StoreID:     order.StoreID,
				Beneficiary: tt.beneficiary,
			})
			if err == nil {
				t.Fatal("Execute() error = nil, want error")
			}
			if payouts.calls != 0 {
				t.Fatalf("Payment payout calls = %d, want 0", payouts.calls)
			}
		})
	}
}

func TestInitiatePayoutPropagatesPaymentErrorAndRejectsNilResponse(t *testing.T) {
	paymentErr := errors.New("Payment service unavailable")
	tests := []struct {
		name    string
		client  *initiatePayoutClientMock
		wantErr string
	}{
		{
			name:    "Payment service error",
			client:  &initiatePayoutClientMock{err: paymentErr},
			wantErr: "Payment service unavailable",
		},
		{
			name:    "nil response",
			client:  &initiatePayoutClientMock{},
			wantErr: "nil payout response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := newReceivedOrder(t)
			uc := use_cases.NewInitiatePayout(
				&initiatePaymentOrderRepository{order: order},
				tt.client,
				initiatePaymentLoggerMock{},
				initiatePaymentMetricsMock{},
			)

			_, err := uc.Execute(context.Background(), use_cases.InitiatePayoutInput{
				OrderID:     order.ID,
				StoreID:     order.StoreID,
				Beneficiary: validPayoutBeneficiary(),
			})
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Execute() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestInitiatePayoutRejectsInvalidOrderPrices(t *testing.T) {
	order := newReceivedOrder(t)
	order.PriceProduct = -1

	payouts := &initiatePayoutClientMock{}
	uc := use_cases.NewInitiatePayout(
		&initiatePaymentOrderRepository{order: order},
		payouts,
		initiatePaymentLoggerMock{},
		initiatePaymentMetricsMock{},
	)

	_, err := uc.Execute(context.Background(), use_cases.InitiatePayoutInput{
		OrderID:     order.ID,
		StoreID:     order.StoreID,
		Beneficiary: validPayoutBeneficiary(),
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want invalid price error")
	}
	if payouts.calls != 0 {
		t.Fatalf("Payment payout calls = %d, want 0", payouts.calls)
	}
}

func TestInitiatePayoutRejectsMismatchedStore(t *testing.T) {
	order := newReceivedOrder(t)
	payouts := &initiatePayoutClientMock{
		response: &ports.InitiatePayoutResponse{PayoutID: "payout-id"},
	}

	uc := use_cases.NewInitiatePayout(
		&initiatePaymentOrderRepository{order: order},
		payouts,
		initiatePaymentLoggerMock{},
		initiatePaymentMetricsMock{},
	)

	_, err := uc.Execute(context.Background(), use_cases.InitiatePayoutInput{
		OrderID:     order.ID,
		StoreID:     uuid.New(),
		Beneficiary: validPayoutBeneficiary(),
	})
	if err == nil || !strings.Contains(err.Error(), "does not belong to the supplied store") {
		t.Fatalf("Execute() error = %v, want store ownership error", err)
	}

	if payouts.calls != 0 {
		t.Fatalf("Payment payout calls = %d, want 0", payouts.calls)
	}
}

func TestInitiatePayoutRejectsMissingStoreID(t *testing.T) {
	order := newReceivedOrder(t)
	payouts := &initiatePayoutClientMock{}

	uc := use_cases.NewInitiatePayout(
		&initiatePaymentOrderRepository{order: order},
		payouts,
		initiatePaymentLoggerMock{},
		initiatePaymentMetricsMock{},
	)

	_, err := uc.Execute(context.Background(), use_cases.InitiatePayoutInput{
		OrderID:     order.ID,
		Beneficiary: validPayoutBeneficiary(),
	})
	if err == nil || !strings.Contains(err.Error(), "store ID is required") {
		t.Fatalf("Execute() error = %v, want missing store ID error", err)
	}

	if payouts.calls != 0 {
		t.Fatalf("Payment payout calls = %d, want 0", payouts.calls)
	}
}
