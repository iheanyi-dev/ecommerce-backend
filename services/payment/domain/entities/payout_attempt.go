package entities

import (
	"strings"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/errors"
)

type PayoutAttempt struct {
	id                string
	payoutID          string
	provider          string
	providerReference string
	status            enums.PayoutAttemptStatus
	failureReason     string
	createdAt         time.Time
	updatedAt         time.Time
}

func NewPayoutAttempt(payoutID, provider string) (*PayoutAttempt, error) {
	payoutID = strings.TrimSpace(payoutID)
	provider = strings.TrimSpace(provider)

	if payoutID == "" || provider == "" {
		return nil, domainerrors.ErrInvalidPayoutAttempt
	}

	id, err := newUUID()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	return &PayoutAttempt{
		id:        id,
		payoutID:  payoutID,
		provider:  provider,
		status:    enums.PayoutAttemptStatusPending,
		createdAt: now,
		updatedAt: now,
	}, nil
}

func (a *PayoutAttempt) ID() string {
	return a.id
}

func (a *PayoutAttempt) PayoutID() string {
	return a.payoutID
}

func (a *PayoutAttempt) Provider() string {
	return a.provider
}

func (a *PayoutAttempt) ProviderReference() string {
	return a.providerReference
}

func (a *PayoutAttempt) Status() enums.PayoutAttemptStatus {
	return a.status
}

func (a *PayoutAttempt) FailureReason() string {
	return a.failureReason
}

func (a *PayoutAttempt) CreatedAt() time.Time {
	return a.createdAt
}

func (a *PayoutAttempt) UpdatedAt() time.Time {
	return a.updatedAt
}

func (a *PayoutAttempt) SetProviderReference(reference string) error {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return domainerrors.ErrInvalidPayoutAttempt
	}

	if a.providerReference != "" {
		if a.providerReference == reference {
			return nil
		}
		return domainerrors.ErrInvalidPayoutAttempt
	}

	if a.status == enums.PayoutAttemptStatusSucceeded ||
		a.status == enums.PayoutAttemptStatusFailed {
		return domainerrors.ErrInvalidPayoutAttemptTransition
	}

	a.providerReference = reference
	a.touch()
	return nil
}

func (a *PayoutAttempt) MarkProcessing() error {
	if a.status != enums.PayoutAttemptStatusPending {
		return domainerrors.ErrInvalidPayoutAttemptTransition
	}

	a.status = enums.PayoutAttemptStatusProcessing
	a.touch()
	return nil
}

func (a *PayoutAttempt) MarkRequiresReconciliation() error {
	if a.status != enums.PayoutAttemptStatusProcessing {
		return domainerrors.ErrInvalidPayoutAttemptTransition
	}

	a.status = enums.PayoutAttemptStatusRequiresReconciliation
	a.touch()
	return nil
}

func (a *PayoutAttempt) MarkSucceeded() error {
	if a.status != enums.PayoutAttemptStatusProcessing &&
		a.status != enums.PayoutAttemptStatusRequiresReconciliation {
		return domainerrors.ErrInvalidPayoutAttemptTransition
	}

	a.status = enums.PayoutAttemptStatusSucceeded
	a.failureReason = ""
	a.touch()
	return nil
}

func (a *PayoutAttempt) MarkFailed(reason string) error {
	if a.status != enums.PayoutAttemptStatusProcessing &&
		a.status != enums.PayoutAttemptStatusRequiresReconciliation {
		return domainerrors.ErrInvalidPayoutAttemptTransition
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		return domainerrors.ErrInvalidPayoutAttempt
	}

	a.status = enums.PayoutAttemptStatusFailed
	a.failureReason = reason
	a.touch()
	return nil
}

func (a *PayoutAttempt) touch() {
	a.updatedAt = time.Now().UTC()
}
