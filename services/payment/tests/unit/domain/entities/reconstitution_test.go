package entities_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/value_objects"
)

const (
	persistedPaymentID        = "11111111-1111-4111-8111-111111111111"
	persistedPaymentAttemptID = "22222222-2222-4222-8222-222222222222"
	persistedPayoutID         = "33333333-3333-4333-8333-333333333333"
	persistedPayoutAttemptID  = "44444444-4444-4444-8444-444444444444"
	persistedPayerID          = "55555555-5555-4555-8555-555555555555"
	persistedStoreID          = "66666666-6666-4666-8666-666666666666"
	persistedOrderID          = "77777777-7777-4777-8777-777777777777"
)

func persistedTimes() (time.Time, time.Time) {
	createdAt := time.Date(2026, 10, 30, 10, 0, 0, 0, time.UTC)
	return createdAt, createdAt.Add(time.Minute)
}

func persistedMoney(t *testing.T) value_objects.Money {
	t.Helper()

	money, err := value_objects.NewMoney(125000, "NGN")
	if err != nil {
		t.Fatalf("create money: %v", err)
	}
	return money
}

func persistedBusinessReference(t *testing.T) value_objects.BusinessReference {
	t.Helper()

	reference, err := value_objects.NewBusinessReference("order", persistedOrderID)
	if err != nil {
		t.Fatalf("create business reference: %v", err)
	}
	return reference
}

func TestReconstitutePayment(t *testing.T) {
	createdAt, updatedAt := persistedTimes()
	money := persistedMoney(t)
	businessReference := persistedBusinessReference(t)

	payment, err := entities.ReconstitutePayment(
		persistedPaymentID,
		persistedPaymentAttemptID,
		"pay_existing_reference",
		persistedPayerID,
		enums.PaymentPurposeOrder,
		businessReference,
		money,
		enums.PaymentStatusProcessing,
		createdAt,
		updatedAt,
	)
	if err != nil {
		t.Fatalf("reconstitute payment: %v", err)
	}

	if payment.ID() != persistedPaymentID ||
		payment.AttemptID() != persistedPaymentAttemptID ||
		payment.Reference() != "pay_existing_reference" ||
		payment.PayerID() != persistedPayerID ||
		payment.Purpose() != enums.PaymentPurposeOrder ||
		payment.BusinessReference().Type() != "order" ||
		payment.BusinessReference().ID() != persistedOrderID ||
		payment.Amount().Amount() != money.Amount() ||
		payment.Amount().Currency() != money.Currency() ||
		payment.Status() != enums.PaymentStatusProcessing ||
		!payment.CreatedAt().Equal(createdAt) ||
		!payment.UpdatedAt().Equal(updatedAt) {
		t.Fatal("reconstituted payment did not preserve persisted state")
	}
}

func TestReconstitutePaymentRejectsInvalidState(t *testing.T) {
	createdAt, updatedAt := persistedTimes()
	money := persistedMoney(t)
	reference := persistedBusinessReference(t)

	tests := []struct {
		name       string
		id         string
		attemptID  string
		paymentRef string
		payerID    string
		purpose    enums.PaymentPurpose
		business   value_objects.BusinessReference
		amount     value_objects.Money
		status     enums.PaymentStatus
		createdAt  time.Time
		updatedAt  time.Time
	}{
		{
			name:       "invalid payment ID",
			id:         "invalid",
			attemptID:  persistedPaymentAttemptID,
			paymentRef: "pay_reference",
			payerID:    persistedPayerID,
			purpose:    enums.PaymentPurposeOrder,
			business:   reference,
			amount:     money,
			status:     enums.PaymentStatusPending,
			createdAt:  createdAt,
			updatedAt:  updatedAt,
		},
		{
			name:       "invalid attempt ID",
			id:         persistedPaymentID,
			attemptID:  "invalid",
			paymentRef: "pay_reference",
			payerID:    persistedPayerID,
			purpose:    enums.PaymentPurposeOrder,
			business:   reference,
			amount:     money,
			status:     enums.PaymentStatusPending,
			createdAt:  createdAt,
			updatedAt:  updatedAt,
		},
		{
			name:      "missing payment reference",
			id:        persistedPaymentID,
			attemptID: persistedPaymentAttemptID,
			payerID:   persistedPayerID,
			purpose:   enums.PaymentPurposeOrder,
			business:  reference,
			amount:    money,
			status:    enums.PaymentStatusPending,
			createdAt: createdAt,
			updatedAt: updatedAt,
		},
		{
			name:       "missing payer",
			id:         persistedPaymentID,
			attemptID:  persistedPaymentAttemptID,
			paymentRef: "pay_reference",
			purpose:    enums.PaymentPurposeOrder,
			business:   reference,
			amount:     money,
			status:     enums.PaymentStatusPending,
			createdAt:  createdAt,
			updatedAt:  updatedAt,
		},
		{
			name:       "invalid purpose",
			id:         persistedPaymentID,
			attemptID:  persistedPaymentAttemptID,
			paymentRef: "pay_reference",
			payerID:    persistedPayerID,
			purpose:    "unknown",
			business:   reference,
			amount:     money,
			status:     enums.PaymentStatusPending,
			createdAt:  createdAt,
			updatedAt:  updatedAt,
		},
		{
			name:       "zero money",
			id:         persistedPaymentID,
			attemptID:  persistedPaymentAttemptID,
			paymentRef: "pay_reference",
			payerID:    persistedPayerID,
			purpose:    enums.PaymentPurposeOrder,
			business:   reference,
			status:     enums.PaymentStatusPending,
			createdAt:  createdAt,
			updatedAt:  updatedAt,
		},
		{
			name:       "invalid status",
			id:         persistedPaymentID,
			attemptID:  persistedPaymentAttemptID,
			paymentRef: "pay_reference",
			payerID:    persistedPayerID,
			purpose:    enums.PaymentPurposeOrder,
			business:   reference,
			amount:     money,
			status:     "unknown",
			createdAt:  createdAt,
			updatedAt:  updatedAt,
		},
		{
			name:       "zero creation time",
			id:         persistedPaymentID,
			attemptID:  persistedPaymentAttemptID,
			paymentRef: "pay_reference",
			payerID:    persistedPayerID,
			purpose:    enums.PaymentPurposeOrder,
			business:   reference,
			amount:     money,
			status:     enums.PaymentStatusPending,
			updatedAt:  updatedAt,
		},
		{
			name:       "updated before created",
			id:         persistedPaymentID,
			attemptID:  persistedPaymentAttemptID,
			paymentRef: "pay_reference",
			payerID:    persistedPayerID,
			purpose:    enums.PaymentPurposeOrder,
			business:   reference,
			amount:     money,
			status:     enums.PaymentStatusPending,
			createdAt:  updatedAt,
			updatedAt:  createdAt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := entities.ReconstitutePayment(
				tt.id, tt.attemptID, tt.paymentRef, tt.payerID,
				tt.purpose, tt.business, tt.amount, tt.status,
				tt.createdAt, tt.updatedAt,
			)
			if err == nil {
				t.Fatal("expected invalid persisted payment to be rejected")
			}
		})
	}
}

func TestReconstitutePaymentAttempt(t *testing.T) {
	createdAt, updatedAt := persistedTimes()
	expiresAt := updatedAt.Add(time.Hour)
	metadata := json.RawMessage(`{"channel":"card"}`)

	attempt, err := entities.ReconstitutePaymentAttempt(
		persistedPaymentAttemptID,
		persistedPaymentID,
		"paystack",
		"provider-reference-123",
		"customer@example.com",
		"https://checkout.example.com/session",
		&expiresAt,
		metadata,
		enums.PaymentAttemptStatusProcessing,
		"",
		createdAt,
		updatedAt,
	)
	if err != nil {
		t.Fatalf("reconstitute payment attempt: %v", err)
	}

	if attempt.ID() != persistedPaymentAttemptID ||
		attempt.PaymentID() != persistedPaymentID ||
		attempt.Provider() != "paystack" ||
		attempt.ProviderReference() != "provider-reference-123" ||
		attempt.CustomerEmail() != "customer@example.com" ||
		attempt.CheckoutURL() != "https://checkout.example.com/session" ||
		attempt.Status() != enums.PaymentAttemptStatusProcessing ||
		!attempt.ExpiresAt().Equal(expiresAt) ||
		string(attempt.Metadata()) != string(metadata) ||
		!attempt.CreatedAt().Equal(createdAt) ||
		!attempt.UpdatedAt().Equal(updatedAt) {
		t.Fatal("reconstituted payment attempt did not preserve persisted state")
	}

	metadata[0] = 'X'
	if string(attempt.Metadata()) != `{"channel":"card"}` {
		t.Fatal("attempt metadata must not share the caller's backing array")
	}
}

func TestReconstitutePaymentAttemptRejectsInvalidState(t *testing.T) {
	createdAt, updatedAt := persistedTimes()
	moneylessMetadata := json.RawMessage(`{"valid":true}`)

	tests := []struct {
		name          string
		id            string
		paymentID     string
		provider      string
		providerRef   string
		status        enums.PaymentAttemptStatus
		failureReason string
		metadata      json.RawMessage
		createdAt     time.Time
		updatedAt     time.Time
	}{
		{
			name:      "invalid ID",
			id:        "invalid",
			paymentID: persistedPaymentID,
			provider:  "paystack",
			status:    enums.PaymentAttemptStatusPending,
			metadata:  moneylessMetadata,
			createdAt: createdAt,
			updatedAt: updatedAt,
		},
		{
			name:      "invalid payment ID",
			id:        persistedPaymentAttemptID,
			paymentID: "invalid",
			provider:  "paystack",
			status:    enums.PaymentAttemptStatusPending,
			metadata:  moneylessMetadata,
			createdAt: createdAt,
			updatedAt: updatedAt,
		},
		{
			name:      "missing provider",
			id:        persistedPaymentAttemptID,
			paymentID: persistedPaymentID,
			status:    enums.PaymentAttemptStatusPending,
			metadata:  moneylessMetadata,
			createdAt: createdAt,
			updatedAt: updatedAt,
		},
		{
			name:      "invalid status",
			id:        persistedPaymentAttemptID,
			paymentID: persistedPaymentID,
			provider:  "paystack",
			status:    "unknown",
			metadata:  moneylessMetadata,
			createdAt: createdAt,
			updatedAt: updatedAt,
		},
		{
			name:      "invalid metadata",
			id:        persistedPaymentAttemptID,
			paymentID: persistedPaymentID,
			provider:  "paystack",
			status:    enums.PaymentAttemptStatusPending,
			metadata:  json.RawMessage(`{"broken"`),
			createdAt: createdAt,
			updatedAt: updatedAt,
		},
		{
			name:      "failed attempt without failure reason",
			id:        persistedPaymentAttemptID,
			paymentID: persistedPaymentID,
			provider:  "paystack",
			status:    enums.PaymentAttemptStatusFailed,
			metadata:  moneylessMetadata,
			createdAt: createdAt,
			updatedAt: updatedAt,
		},
		{
			name:      "updated before created",
			id:        persistedPaymentAttemptID,
			paymentID: persistedPaymentID,
			provider:  "paystack",
			status:    enums.PaymentAttemptStatusPending,
			metadata:  moneylessMetadata,
			createdAt: updatedAt,
			updatedAt: createdAt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := entities.ReconstitutePaymentAttempt(
				tt.id, tt.paymentID, tt.provider, tt.providerRef,
				"", "", nil, tt.metadata, tt.status, tt.failureReason,
				tt.createdAt, tt.updatedAt,
			)
			if err == nil {
				t.Fatal("expected invalid persisted payment attempt to be rejected")
			}
		})
	}
}

func TestReconstitutePayout(t *testing.T) {
	createdAt, updatedAt := persistedTimes()
	money := persistedMoney(t)
	reference := persistedBusinessReference(t)

	beneficiary := entities.PayoutBeneficiary{
		AccountName:   "Example Store",
		AccountNumber: "0123456789",
		BankCode:      "058",
		BankName:      "Example Bank",
		CountryCode:   "NG",
	}

	payout, err := entities.ReconstitutePayout(
		persistedPayoutID,
		persistedPayoutAttemptID,
		"spay_existing_reference",
		reference,
		"Order settlement",
		beneficiary,
		persistedStoreID,
		persistedOrderID,
		money,
		enums.PayoutStatusProcessing,
		createdAt,
		updatedAt,
	)
	if err != nil {
		t.Fatalf("reconstitute payout: %v", err)
	}

	if payout.ID() != persistedPayoutID ||
		payout.AttemptID() != persistedPayoutAttemptID ||
		payout.Reference() != "spay_existing_reference" ||
		payout.BusinessReference().Type() != "order" ||
		payout.BusinessReference().ID() != persistedOrderID ||
		payout.Reason() != "Order settlement" ||
		payout.Beneficiary() != beneficiary ||
		payout.StoreID() != persistedStoreID ||
		payout.OrderID() != persistedOrderID ||
		payout.Amount().Amount() != money.Amount() ||
		payout.Status() != enums.PayoutStatusProcessing ||
		!payout.CreatedAt().Equal(createdAt) ||
		!payout.UpdatedAt().Equal(updatedAt) {
		t.Fatal("reconstituted payout did not preserve persisted state")
	}
}

func TestReconstituteLegacyPayoutWithoutBusinessReference(t *testing.T) {
	createdAt, updatedAt := persistedTimes()

	payout, err := entities.ReconstitutePayout(
		persistedPayoutID,
		"",
		"spay_legacy_reference",
		value_objects.BusinessReference{},
		"",
		entities.PayoutBeneficiary{},
		persistedStoreID,
		persistedOrderID,
		persistedMoney(t),
		enums.PayoutStatusPending,
		createdAt,
		updatedAt,
	)
	if err != nil {
		t.Fatalf("reconstitute legacy payout: %v", err)
	}

	if !payout.BusinessReference().IsZero() ||
		payout.AttemptID() != "" ||
		payout.Beneficiary() != (entities.PayoutBeneficiary{}) {
		t.Fatal("legacy payout optional fields were not preserved")
	}
}

func TestReconstitutePayoutRejectsInvalidState(t *testing.T) {
	createdAt, updatedAt := persistedTimes()
	money := persistedMoney(t)
	reference := persistedBusinessReference(t)

	tests := []struct {
		name      string
		id        string
		attemptID string
		payoutRef string
		storeID   string
		orderID   string
		amount    value_objects.Money
		status    enums.PayoutStatus
		createdAt time.Time
		updatedAt time.Time
	}{
		{
			name:      "invalid ID",
			id:        "invalid",
			payoutRef: "spay_reference",
			storeID:   persistedStoreID,
			orderID:   persistedOrderID,
			amount:    money,
			status:    enums.PayoutStatusPending,
			createdAt: createdAt,
			updatedAt: updatedAt,
		},
		{
			name:      "invalid attempt ID",
			id:        persistedPayoutID,
			attemptID: "invalid",
			payoutRef: "spay_reference",
			storeID:   persistedStoreID,
			orderID:   persistedOrderID,
			amount:    money,
			status:    enums.PayoutStatusPending,
			createdAt: createdAt,
			updatedAt: updatedAt,
		},
		{
			name:      "missing reference",
			id:        persistedPayoutID,
			storeID:   persistedStoreID,
			orderID:   persistedOrderID,
			amount:    money,
			status:    enums.PayoutStatusPending,
			createdAt: createdAt,
			updatedAt: updatedAt,
		},
		{
			name:      "missing store",
			id:        persistedPayoutID,
			payoutRef: "spay_reference",
			orderID:   persistedOrderID,
			amount:    money,
			status:    enums.PayoutStatusPending,
			createdAt: createdAt,
			updatedAt: updatedAt,
		},
		{
			name:      "zero money",
			id:        persistedPayoutID,
			payoutRef: "spay_reference",
			storeID:   persistedStoreID,
			orderID:   persistedOrderID,
			status:    enums.PayoutStatusPending,
			createdAt: createdAt,
			updatedAt: updatedAt,
		},
		{
			name:      "invalid status",
			id:        persistedPayoutID,
			payoutRef: "spay_reference",
			storeID:   persistedStoreID,
			orderID:   persistedOrderID,
			amount:    money,
			status:    "unknown",
			createdAt: createdAt,
			updatedAt: updatedAt,
		},
		{
			name:      "updated before created",
			id:        persistedPayoutID,
			payoutRef: "spay_reference",
			storeID:   persistedStoreID,
			orderID:   persistedOrderID,
			amount:    money,
			status:    enums.PayoutStatusPending,
			createdAt: updatedAt,
			updatedAt: createdAt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := entities.ReconstitutePayout(
				tt.id, tt.attemptID, tt.payoutRef, reference,
				"reason", entities.PayoutBeneficiary{
					AccountName:   "Example Store",
					AccountNumber: "0123456789",
					BankCode:      "058",
					BankName:      "Example Bank",
					CountryCode:   "NG",
				},
				tt.storeID, tt.orderID, tt.amount, tt.status,
				tt.createdAt, tt.updatedAt,
			)
			if err == nil {
				t.Fatal("expected invalid persisted payout to be rejected")
			}
		})
	}
}

func TestReconstitutePayoutAttempt(t *testing.T) {
	createdAt, updatedAt := persistedTimes()

	attempt, err := entities.ReconstitutePayoutAttempt(
		persistedPayoutAttemptID,
		persistedPayoutID,
		"paystack",
		"transfer-reference-123",
		enums.PayoutAttemptStatusRequiresReconciliation,
		"",
		createdAt,
		updatedAt,
	)
	if err != nil {
		t.Fatalf("reconstitute payout attempt: %v", err)
	}

	if attempt.ID() != persistedPayoutAttemptID ||
		attempt.PayoutID() != persistedPayoutID ||
		attempt.Provider() != "paystack" ||
		attempt.ProviderReference() != "transfer-reference-123" ||
		attempt.Status() != enums.PayoutAttemptStatusRequiresReconciliation ||
		!attempt.CreatedAt().Equal(createdAt) ||
		!attempt.UpdatedAt().Equal(updatedAt) {
		t.Fatal("reconstituted payout attempt did not preserve persisted state")
	}
}

func TestReconstitutePayoutAttemptRejectsInvalidState(t *testing.T) {
	createdAt, updatedAt := persistedTimes()

	tests := []struct {
		name          string
		id            string
		payoutID      string
		provider      string
		status        enums.PayoutAttemptStatus
		failureReason string
		createdAt     time.Time
		updatedAt     time.Time
	}{
		{
			name:      "invalid ID",
			id:        "invalid",
			payoutID:  persistedPayoutID,
			provider:  "paystack",
			status:    enums.PayoutAttemptStatusPending,
			createdAt: createdAt,
			updatedAt: updatedAt,
		},
		{
			name:      "invalid payout ID",
			id:        persistedPayoutAttemptID,
			payoutID:  "invalid",
			provider:  "paystack",
			status:    enums.PayoutAttemptStatusPending,
			createdAt: createdAt,
			updatedAt: updatedAt,
		},
		{
			name:      "missing provider",
			id:        persistedPayoutAttemptID,
			payoutID:  persistedPayoutID,
			status:    enums.PayoutAttemptStatusPending,
			createdAt: createdAt,
			updatedAt: updatedAt,
		},
		{
			name:      "invalid status",
			id:        persistedPayoutAttemptID,
			payoutID:  persistedPayoutID,
			provider:  "paystack",
			status:    "unknown",
			createdAt: createdAt,
			updatedAt: updatedAt,
		},
		{
			name:      "failed attempt without reason",
			id:        persistedPayoutAttemptID,
			payoutID:  persistedPayoutID,
			provider:  "paystack",
			status:    enums.PayoutAttemptStatusFailed,
			createdAt: createdAt,
			updatedAt: updatedAt,
		},
		{
			name:      "updated before created",
			id:        persistedPayoutAttemptID,
			payoutID:  persistedPayoutID,
			provider:  "paystack",
			status:    enums.PayoutAttemptStatusPending,
			createdAt: updatedAt,
			updatedAt: createdAt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := entities.ReconstitutePayoutAttempt(
				tt.id, tt.payoutID, tt.provider, "",
				tt.status, tt.failureReason, tt.createdAt, tt.updatedAt,
			)
			if err == nil {
				t.Fatal("expected invalid persisted payout attempt to be rejected")
			}
		})
	}
}
