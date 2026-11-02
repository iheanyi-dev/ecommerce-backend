package entities

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/value_objects"
)

type Payment struct {
	attemptID         string
	id                string
	reference         string
	payerID           string
	purpose           enums.PaymentPurpose
	businessReference value_objects.BusinessReference
	amount            value_objects.Money
	status            enums.PaymentStatus
	createdAt         time.Time
	updatedAt         time.Time
}

func NewPayment(
	payerID string,
	purpose enums.PaymentPurpose,
	businessReference value_objects.BusinessReference,
	amount value_objects.Money,
) (*Payment, error) {
	payerID = strings.TrimSpace(payerID)

	if payerID == "" ||
		!purpose.IsValid() ||
		businessReference.IsZero() ||
		amount.IsZero() {
		return nil, domainerrors.ErrInvalidPayment
	}

	id, err := newUUID()
	if err != nil {
		return nil, fmt.Errorf("generate payment ID: %w", err)
	}

	referenceID, err := newUUID()
	if err != nil {
		return nil, fmt.Errorf("generate payment reference: %w", err)
	}

	now := time.Now().UTC()

	return &Payment{
		id:                id,
		reference:         "pay_" + strings.ReplaceAll(referenceID, "-", ""),
		payerID:           payerID,
		purpose:           purpose,
		businessReference: businessReference,
		amount:            amount,
		status:            enums.PaymentStatusPending,
		createdAt:         now,
		updatedAt:         now,
	}, nil
}

func (p *Payment) ID() string {
	return p.id
}

func (p *Payment) Reference() string {
	return p.reference
}

func (p *Payment) PayerID() string {
	return p.payerID
}

func (p *Payment) Purpose() enums.PaymentPurpose {
	return p.purpose
}

func (p *Payment) BusinessReference() value_objects.BusinessReference {
	return p.businessReference
}

func (p *Payment) Amount() value_objects.Money {
	return p.amount
}

func (p *Payment) Status() enums.PaymentStatus {
	return p.status
}

func (p *Payment) CreatedAt() time.Time {
	return p.createdAt
}

func (p *Payment) UpdatedAt() time.Time {
	return p.updatedAt
}

func (p *Payment) MarkProcessing() error {
	if p.status != enums.PaymentStatusPending {
		return domainerrors.ErrInvalidPaymentTransition
	}

	p.status = enums.PaymentStatusProcessing
	p.touch()
	return nil
}

func (p *Payment) MarkSucceeded() error {
	if p.status != enums.PaymentStatusProcessing {
		return domainerrors.ErrInvalidPaymentTransition
	}

	p.status = enums.PaymentStatusSucceeded
	p.touch()
	return nil
}

func (p *Payment) MarkFailed() error {
	if p.status != enums.PaymentStatusProcessing {
		return domainerrors.ErrInvalidPaymentTransition
	}

	p.status = enums.PaymentStatusFailed
	p.touch()
	return nil
}

// AttemptID returns the current payment attempt UUID, or an empty string if none is linked.
func (p *Payment) AttemptID() string {
	return p.attemptID
}

// SetAttemptID links the payment to its current attempt.
// Attempt IDs must be UUIDs, and an existing link cannot be replaced by a different attempt.
func (p *Payment) SetAttemptID(id string) error {
	if err := validateEntityUUID(id); err != nil {
		return fmt.Errorf("invalid payment attempt ID: %w", err)
	}
	if p.attemptID != "" && p.attemptID != id {
		return fmt.Errorf("payment already has a different current attempt")
	}
	if p.attemptID != id {
		p.attemptID = id
		p.touch()
	}
	return nil
}

func (p *Payment) touch() {
	p.updatedAt = time.Now().UTC()
}

func newUUID() (string, error) {
	var b [16]byte

	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}

	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16],
	), nil
}

// validateEntityUUID validates the canonical UUID representation used by domain IDs.
func validateEntityUUID(value string) error {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' ||
		value[18] != '-' || value[23] != '-' {
		return fmt.Errorf("invalid UUID format")
	}
	for i, r := range value {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return fmt.Errorf("invalid UUID format")
		}
	}
	return nil
}
