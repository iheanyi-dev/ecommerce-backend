package entities

import (
	"fmt"
	"strings"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/value_objects"
)

// Payout represents a transfer to a beneficiary for a business operation.
type Payout struct {
	id                string
	attemptID         string
	reference         string
	businessReference value_objects.BusinessReference
	reason            string
	beneficiary       PayoutBeneficiary
	storeID           string
	orderID           string
	amount            value_objects.Money
	status            enums.PayoutStatus
	createdAt         time.Time
	updatedAt         time.Time
}

// PayoutBeneficiary is a snapshot of the destination supplied for this payout.
// Account details must never be written to application logs.
type PayoutBeneficiary struct {
	AccountName   string
	AccountNumber string
	BankCode      string
	BankName      string
	CountryCode   string
}

// NewPayout creates a payout with an immutable beneficiary snapshot.
func NewPayout(
	businessReference value_objects.BusinessReference,
	storeID string,
	orderID string,
	reason string,
	amount value_objects.Money,
	beneficiary PayoutBeneficiary,
) (*Payout, error) {

	storeID = strings.TrimSpace(storeID)
	orderID = strings.TrimSpace(orderID)
	reason = strings.TrimSpace(reason)
	beneficiary.AccountName = strings.TrimSpace(beneficiary.AccountName)
	beneficiary.AccountNumber = strings.TrimSpace(beneficiary.AccountNumber)
	beneficiary.BankCode = strings.TrimSpace(beneficiary.BankCode)
	beneficiary.BankName = strings.TrimSpace(beneficiary.BankName)
	beneficiary.CountryCode = strings.ToUpper(strings.TrimSpace(beneficiary.CountryCode))

	if businessReference.IsZero() || storeID == "" || orderID == "" ||
		reason == "" || amount.IsZero() ||
		beneficiary.AccountName == "" || beneficiary.AccountNumber == "" ||
		beneficiary.BankCode == "" || beneficiary.BankName == "" ||
		len(beneficiary.CountryCode) != 2 {
		return nil, domainerrors.ErrInvalidPayout
	}

	id, err := newUUID()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()

	return &Payout{
		id:                id,
		reference:         "payou_" + strings.ReplaceAll(id, "-", ""),
		businessReference: businessReference,
		storeID:           storeID,
		orderID:           orderID,
		reason:            reason,
		beneficiary:       beneficiary,
		amount:            amount,
		status:            enums.PayoutStatusPending,
		createdAt:         now,
		updatedAt:         now,
	}, nil
}

func (p *Payout) ID() string {
	return p.id
}

func (p *Payout) Reference() string {
	return p.reference
}

func (p *Payout) BusinessReference() value_objects.BusinessReference {
	return p.businessReference
}

// AttemptID returns the current payout attempt ID, or empty if none is linked.
func (p *Payout) AttemptID() string {
	return p.attemptID
}

// SetAttemptID links an attempt. Replacing an attempt requires a definitive payout failure.
func (p *Payout) SetAttemptID(id string) error {
	if err := validateEntityUUID(id); err != nil {
		return fmt.Errorf("invalid payout attempt ID: %w", err)
	}
	if p.attemptID == id {
		return nil
	}
	if p.attemptID != "" && p.status != enums.PayoutStatusFailed {
		return domainerrors.ErrInvalidPayoutTransition
	}
	p.attemptID = id
	p.touch()
	return nil
}

func (p *Payout) Reason() string {
	return p.reason
}

func (p *Payout) Beneficiary() PayoutBeneficiary {
	return p.beneficiary
}

func (p *Payout) StoreID() string {
	return p.storeID
}

func (p *Payout) OrderID() string {
	return p.orderID
}

func (p *Payout) Amount() value_objects.Money {
	return p.amount
}

func (p *Payout) Status() enums.PayoutStatus {
	return p.status
}

func (p *Payout) CreatedAt() time.Time {
	return p.createdAt
}

func (p *Payout) UpdatedAt() time.Time {
	return p.updatedAt
}

// MarkProcessing starts processing a new payout or a retry after
// a definitive failure. Application-level coordination must ensure
// no unresolved provider attempt exists before a retry is created.
func (p *Payout) MarkProcessing() error {
	if p.status != enums.PayoutStatusPending &&
		p.status != enums.PayoutStatusFailed {
		return domainerrors.ErrInvalidPayoutTransition
	}

	p.status = enums.PayoutStatusProcessing
	p.touch()
	return nil
}

func (p *Payout) MarkSucceeded() error {
	if p.status != enums.PayoutStatusProcessing {
		return domainerrors.ErrInvalidPayoutTransition
	}

	p.status = enums.PayoutStatusSucceeded
	p.touch()
	return nil
}

func (p *Payout) MarkFailed() error {
	if p.status != enums.PayoutStatusProcessing {
		return domainerrors.ErrInvalidPayoutTransition
	}

	p.status = enums.PayoutStatusFailed
	p.touch()
	return nil
}

func (p *Payout) touch() {
	p.updatedAt = time.Now().UTC()
}
