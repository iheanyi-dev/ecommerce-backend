package entities_test

import (
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/value_objects"
)

func newTestPayout(t *testing.T) *entities.Payout {
	t.Helper()

	businessReference, err := value_objects.NewBusinessReference("order", "order-123")
	if err != nil {
		t.Fatal(err)
	}

	amount, err := value_objects.NewMoney(9500, "NGN")
	if err != nil {
		t.Fatal(err)
	}

	payout, err := entities.NewPayout(
		businessReference,
		"store-123",
		"order-123",
		"Vendor settlement",
		amount,
		entities.PayoutBeneficiary{
			AccountName:   "Test Vendor",
			AccountNumber: "0123456789",
			BankCode:      "058",
			BankName:      "Test Bank",
			CountryCode:   "NG",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	return payout
}

func TestNewPayout(t *testing.T) {
	payout := newTestPayout(t)

	if payout.ID() == "" || payout.Reference() == "" {
		t.Error("payout ID and reference must not be empty")
	}
	if payout.StoreID() != "store-123" || payout.OrderID() != "order-123" {
		t.Error("payout business references were not preserved")
	}
	if payout.Amount().Amount() != 9500 || payout.Amount().Currency() != "NGN" {
		t.Error("payout amount was not preserved")
	}
	if payout.Status() != enums.PayoutStatusPending {
		t.Errorf("Status() = %q, want pending", payout.Status())
	}
	if payout.Reason() != "Vendor settlement" {
		t.Errorf("Reason() = %q, want Vendor settlement", payout.Reason())
	}
	if payout.Beneficiary().AccountNumber != "0123456789" {
		t.Error("beneficiary snapshot was not preserved")
	}
	if payout.CreatedAt().IsZero() || !payout.CreatedAt().Equal(payout.UpdatedAt()) {
		t.Error("initial timestamps should be set and equal")
	}
}

func TestNewPayoutRejectsInvalidInput(t *testing.T) {
	businessReference, err := value_objects.NewBusinessReference("order", "order-123")
	if err != nil {
		t.Fatal(err)
	}

	amount, err := value_objects.NewMoney(100, "NGN")
	if err != nil {
		t.Fatal(err)
	}

	beneficiary := entities.PayoutBeneficiary{
		AccountName:   "Test Vendor",
		AccountNumber: "0123456789",
		BankCode:      "058",
		BankName:      "Test Bank",
		CountryCode:   "NG",
	}

	tests := []struct {
		name              string
		businessReference value_objects.BusinessReference
		storeID           string
		orderID           string
		reason            string
		amount            value_objects.Money
		beneficiary       entities.PayoutBeneficiary
	}{
		{
			name:        "zero business reference",
			storeID:     "store-1",
			orderID:     "order-1",
			reason:      "Settlement",
			amount:      amount,
			beneficiary: beneficiary,
		},
		{
			name:              "empty store ID",
			businessReference: businessReference,
			storeID:           "",
			orderID:           "order-1",
			reason:            "Settlement",
			amount:            amount,
			beneficiary:       beneficiary,
		},
		{
			name:              "empty order ID",
			businessReference: businessReference,
			storeID:           "store-1",
			orderID:           " ",
			reason:            "Settlement",
			amount:            amount,
			beneficiary:       beneficiary,
		},
		{
			name:              "empty reason",
			businessReference: businessReference,
			storeID:           "store-1",
			orderID:           "order-1",
			reason:            " ",
			amount:            amount,
			beneficiary:       beneficiary,
		},
		{
			name:              "zero amount",
			businessReference: businessReference,
			storeID:           "store-1",
			orderID:           "order-1",
			reason:            "Settlement",
			beneficiary:       beneficiary,
		},
		{
			name:              "incomplete beneficiary",
			businessReference: businessReference,
			storeID:           "store-1",
			orderID:           "order-1",
			reason:            "Settlement",
			amount:            amount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := entities.NewPayout(
				tt.businessReference,
				tt.storeID,
				tt.orderID,
				tt.reason,
				tt.amount,
				tt.beneficiary,
			)
			if !errors.Is(err, domainerrors.ErrInvalidPayout) {
				t.Errorf("error = %v, want ErrInvalidPayout", err)
			}
		})
	}
}

func TestPayoutLifecycleAndRetry(t *testing.T) {
	payout := newTestPayout(t)

	if err := payout.MarkProcessing(); err != nil {
		t.Fatal(err)
	}
	if err := payout.MarkFailed(); err != nil {
		t.Fatal(err)
	}
	if payout.Status() != enums.PayoutStatusFailed {
		t.Fatalf("Status() = %q, want failed", payout.Status())
	}

	// A retry reuses the payout but must create a separate attempt.
	if err := payout.MarkProcessing(); err != nil {
		t.Fatalf("retry after definitive failure should be allowed: %v", err)
	}
	if err := payout.MarkSucceeded(); err != nil {
		t.Fatal(err)
	}
	if payout.Status() != enums.PayoutStatusSucceeded {
		t.Errorf("Status() = %q, want succeeded", payout.Status())
	}
	if err := payout.MarkProcessing(); !errors.Is(err, domainerrors.ErrInvalidPayoutTransition) {
		t.Errorf("retry after success error = %v", err)
	}
}

func TestPayoutRejectsInvalidTransitionsWithoutMutation(t *testing.T) {
	payout := newTestPayout(t)
	before := payout.UpdatedAt()

	if err := payout.MarkSucceeded(); !errors.Is(err, domainerrors.ErrInvalidPayoutTransition) {
		t.Fatalf("error = %v, want ErrInvalidPayoutTransition", err)
	}
	if err := payout.MarkFailed(); !errors.Is(err, domainerrors.ErrInvalidPayoutTransition) {
		t.Fatalf("error = %v, want ErrInvalidPayoutTransition", err)
	}
	if payout.Status() != enums.PayoutStatusPending {
		t.Error("rejected transition changed payout status")
	}
	if !payout.UpdatedAt().Equal(before) {
		t.Error("rejected transition changed UpdatedAt")
	}
}
