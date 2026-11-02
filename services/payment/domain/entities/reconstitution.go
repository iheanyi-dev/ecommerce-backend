package entities

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/value_objects"
)

// ReconstitutePayment rebuilds a Payment from persisted state without
// generating new identifiers or changing its lifecycle state.
func ReconstitutePayment(
	id, attemptID, reference, payerID string,
	purpose enums.PaymentPurpose,
	businessReference value_objects.BusinessReference,
	amount value_objects.Money,
	status enums.PaymentStatus,
	createdAt, updatedAt time.Time,
) (*Payment, error) {
	id = strings.TrimSpace(id)
	attemptID = strings.TrimSpace(attemptID)
	reference = strings.TrimSpace(reference)
	payerID = strings.TrimSpace(payerID)

	if validateEntityUUID(id) != nil ||
		(attemptID != "" && validateEntityUUID(attemptID) != nil) ||
		validateEntityUUID(payerID) != nil ||
		reference == "" || !purpose.IsValid() ||
		businessReference.IsZero() || amount.IsZero() ||
		!status.IsValid() ||
		!validPersistedTimestamps(createdAt, updatedAt) {
		return nil, fmt.Errorf("%w: invalid persisted state", domainerrors.ErrInvalidPayment)
	}

	return &Payment{
		id:                id,
		attemptID:         attemptID,
		reference:         reference,
		payerID:           payerID,
		purpose:           purpose,
		businessReference: businessReference,
		amount:            amount,
		status:            status,
		createdAt:         createdAt,
		updatedAt:         updatedAt,
	}, nil
}

// ReconstitutePaymentAttempt rebuilds a PaymentAttempt from persisted state.
// Metadata and expiry timestamps are copied to avoid aliasing caller data.
func ReconstitutePaymentAttempt(
	id, paymentID, provider, providerReference,
	customerEmail, checkoutURL string,
	expiresAt *time.Time,
	metadata json.RawMessage,
	status enums.PaymentAttemptStatus,
	failureReason string,
	createdAt, updatedAt time.Time,
) (*PaymentAttempt, error) {
	id = strings.TrimSpace(id)
	paymentID = strings.TrimSpace(paymentID)
	provider = strings.TrimSpace(provider)
	providerReference = strings.TrimSpace(providerReference)
	customerEmail = strings.TrimSpace(customerEmail)
	checkoutURL = strings.TrimSpace(checkoutURL)
	failureReason = strings.TrimSpace(failureReason)

	if validateEntityUUID(id) != nil ||
		validateEntityUUID(paymentID) != nil ||
		provider == "" || !status.IsValid() ||
		!validPersistedTimestamps(createdAt, updatedAt) ||
		(len(metadata) > 0 && !json.Valid(metadata)) ||
		(status == enums.PaymentAttemptStatusFailed && failureReason == "") ||
		(status != enums.PaymentAttemptStatusFailed && failureReason != "") {
		return nil, fmt.Errorf("%w: invalid persisted state", domainerrors.ErrInvalidPaymentAttempt)
	}

	if expiresAt != nil && expiresAt.IsZero() {
		return nil, fmt.Errorf("%w: invalid checkout expiry", domainerrors.ErrInvalidPaymentAttempt)
	}

	var expiryCopy *time.Time
	if expiresAt != nil {
		value := *expiresAt
		expiryCopy = &value
	}

	return &PaymentAttempt{
		id:                id,
		paymentID:         paymentID,
		provider:          provider,
		providerReference: providerReference,
		customerEmail:     customerEmail,
		checkoutURL:       checkoutURL,
		expiresAt:         expiryCopy,
		metadata:          append(json.RawMessage(nil), metadata...),
		status:            status,
		failureReason:     failureReason,
		createdAt:         createdAt,
		updatedAt:         updatedAt,
	}, nil
}

// ReconstitutePayout rebuilds a payout, including legacy payouts that have
// no business reference or beneficiary snapshot.
func ReconstitutePayout(
	id, attemptID, reference string,
	businessReference value_objects.BusinessReference,
	reason string,
	beneficiary PayoutBeneficiary,
	storeID, orderID string,
	amount value_objects.Money,
	status enums.PayoutStatus,
	createdAt, updatedAt time.Time,
) (*Payout, error) {
	id = strings.TrimSpace(id)
	attemptID = strings.TrimSpace(attemptID)
	reference = strings.TrimSpace(reference)
	reason = strings.TrimSpace(reason)
	storeID = strings.TrimSpace(storeID)
	orderID = strings.TrimSpace(orderID)

	beneficiary.AccountName = strings.TrimSpace(beneficiary.AccountName)
	beneficiary.AccountNumber = strings.TrimSpace(beneficiary.AccountNumber)
	beneficiary.BankCode = strings.TrimSpace(beneficiary.BankCode)
	beneficiary.BankName = strings.TrimSpace(beneficiary.BankName)
	beneficiary.CountryCode = strings.ToUpper(strings.TrimSpace(beneficiary.CountryCode))

	beneficiaryEmpty := beneficiary == (PayoutBeneficiary{})
	beneficiaryComplete := beneficiary.AccountName != "" &&
		beneficiary.AccountNumber != "" &&
		beneficiary.BankCode != "" &&
		beneficiary.BankName != "" &&
		len(beneficiary.CountryCode) == 2

	if validateEntityUUID(id) != nil ||
		(attemptID != "" && validateEntityUUID(attemptID) != nil) ||
		validateEntityUUID(storeID) != nil ||
		validateEntityUUID(orderID) != nil ||
		reference == "" || amount.IsZero() ||
		!status.IsValid() ||
		(!beneficiaryEmpty && !beneficiaryComplete) ||
		(!beneficiaryEmpty && businessReference.IsZero()) ||
		(!beneficiaryEmpty && reason == "") ||
		!validPersistedTimestamps(createdAt, updatedAt) {
		return nil, fmt.Errorf("%w: invalid persisted state", domainerrors.ErrInvalidPayout)
	}

	return &Payout{
		id:                id,
		attemptID:         attemptID,
		reference:         reference,
		businessReference: businessReference,
		reason:            reason,
		beneficiary:       beneficiary,
		storeID:           storeID,
		orderID:           orderID,
		amount:            amount,
		status:            status,
		createdAt:         createdAt,
		updatedAt:         updatedAt,
	}, nil
}

// ReconstitutePayoutAttempt rebuilds a payout attempt from persisted state.
func ReconstitutePayoutAttempt(
	id, payoutID, provider, providerReference string,
	status enums.PayoutAttemptStatus,
	failureReason string,
	createdAt, updatedAt time.Time,
) (*PayoutAttempt, error) {
	id = strings.TrimSpace(id)
	payoutID = strings.TrimSpace(payoutID)
	provider = strings.TrimSpace(provider)
	providerReference = strings.TrimSpace(providerReference)
	failureReason = strings.TrimSpace(failureReason)

	if validateEntityUUID(id) != nil ||
		validateEntityUUID(payoutID) != nil ||
		provider == "" || !status.IsValid() ||
		!validPersistedTimestamps(createdAt, updatedAt) ||
		(status == enums.PayoutAttemptStatusFailed && failureReason == "") ||
		(status != enums.PayoutAttemptStatusFailed && failureReason != "") {
		return nil, fmt.Errorf("%w: invalid persisted state", domainerrors.ErrInvalidPayoutAttempt)
	}

	return &PayoutAttempt{
		id:                id,
		payoutID:          payoutID,
		provider:          provider,
		providerReference: providerReference,
		status:            status,
		failureReason:     failureReason,
		createdAt:         createdAt,
		updatedAt:         updatedAt,
	}, nil
}

func validPersistedTimestamps(createdAt, updatedAt time.Time) bool {
	return !createdAt.IsZero() &&
		!updatedAt.IsZero() &&
		!updatedAt.Before(createdAt)
}
