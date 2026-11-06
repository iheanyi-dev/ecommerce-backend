package use_cases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
)

type InitiatePayoutOutput struct {
	PayoutID      string
	AttemptID     string
	Reference     string
	Status        enums.PayoutStatus
	AttemptStatus enums.PayoutAttemptStatus
	Provider      string
	Amount        int64
	Currency      string
}

type InitiatePayoutUseCase struct {
	payments     ports.PaymentRepository
	payouts      ports.PayoutRepository
	attempts     ports.PayoutAttemptRepository
	provider     ports.PayoutProvider
	outbox       ports.PaymentOutboxRepository
	uow          ports.UnitOfWork
	providerName string
}

func NewInitiatePayoutUseCase(
	payments ports.PaymentRepository,
	payouts ports.PayoutRepository,
	attempts ports.PayoutAttemptRepository,
	provider ports.PayoutProvider,
	outbox ports.PaymentOutboxRepository,
	uow ports.UnitOfWork,
	providerName string,
) (*InitiatePayoutUseCase, error) {
	if payments == nil || payouts == nil || attempts == nil ||
		provider == nil || outbox == nil || uow == nil {
		return nil, fmt.Errorf("payout initiation dependencies are required")
	}
	if strings.TrimSpace(providerName) == "" {
		return nil, fmt.Errorf("provider name is required")
	}
	return &InitiatePayoutUseCase{
		payments:     payments,
		payouts:      payouts,
		attempts:     attempts,
		provider:     provider,
		outbox:       outbox,
		uow:          uow,
		providerName: strings.TrimSpace(providerName),
	}, nil
}

func (uc *InitiatePayoutUseCase) Execute(
	ctx context.Context,
	input ports.PayoutRequest,
) (*InitiatePayoutOutput, error) {
	if input.BusinessReference.IsZero() ||
		strings.TrimSpace(input.StoreID) == "" ||
		strings.TrimSpace(input.OrderID) == "" ||
		strings.TrimSpace(input.Reason) == "" ||
		input.Amount.IsZero() {
		return nil, fmt.Errorf("invalid payout request")
	}

	// A vendor payout is only eligible after the corresponding customer payment
	// has succeeded. Never create a payout before verifying this condition.
	payment, err := uc.payments.GetByBusinessReference(ctx, input.BusinessReference)
	if err != nil {
		if errors.Is(err, application.ErrPaymentNotFound) {
			return nil, fmt.Errorf("find payment for payout: %w", application.ErrPaymentNotFound)
		}
		return nil, fmt.Errorf("find payment for payout: %w", err)
	}
	if payment.Status() != enums.PaymentStatusSucceeded {
		return nil, fmt.Errorf("customer payment must succeed before payout: payment status %q", payment.Status())
	}
	payout, err := uc.payouts.GetByBusinessReference(ctx, input.BusinessReference)
	if err == nil && payout != nil {
		if !samePayoutRequest(payout, input) {
			return nil, application.ErrPayoutIdempotencyConflict
		}
		return uc.reconcileExisting(ctx, payout)
	}
	if err != nil && !errors.Is(err, application.ErrPayoutNotFound) {
		return nil, fmt.Errorf("find existing payout: %w", err)
	}

	payout, err = entities.NewPayout(
		input.BusinessReference,
		input.StoreID,
		input.OrderID,
		input.Reason,
		input.Amount,
		input.Beneficiary,
	)
	if err != nil {
		return nil, fmt.Errorf("create payout: %w", err)
	}

	attempt, err := entities.NewPayoutAttempt(payout.ID(), uc.providerName)
	if err != nil {
		return nil, fmt.Errorf("create payout attempt: %w", err)
	}
	if err := payout.SetAttemptID(attempt.ID()); err != nil {
		return nil, fmt.Errorf("link payout attempt: %w", err)
	}

	// Persist before initiating the external transfer. The attempt UUID is
	// the stable Paystack reference used to reconcile an ambiguous response.
	if err := uc.uow.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.payouts.Save(txCtx, payout); err != nil {
			return fmt.Errorf("save payout: %w", err)
		}
		if err := uc.attempts.Save(txCtx, attempt); err != nil {
			return fmt.Errorf("save payout attempt: %w", err)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("persist payout before provider initiation: %w", err)
	}

	result, err := uc.provider.InitiatePayout(ctx, ports.PayoutInitiationRequest{
		Payout:  payout,
		Attempt: attempt,
	})
	if err != nil {
		// A timeout is ambiguous: the provider may already have initiated
		// the transfer. Persist a reconciliation state instead of retrying.
		if transitionErr := attempt.MarkProcessing(); transitionErr != nil {
			return nil, fmt.Errorf("mark payout attempt processing after provider error: %w", transitionErr)
		}
		if transitionErr := attempt.MarkRequiresReconciliation(); transitionErr != nil {
			return nil, fmt.Errorf("mark payout attempt for reconciliation: %w", transitionErr)
		}
		if transitionErr := payout.MarkProcessing(); transitionErr != nil {
			return nil, fmt.Errorf("mark payout processing after provider error: %w", transitionErr)
		}
		if persistErr := uc.uow.WithinTransaction(ctx, func(txCtx context.Context) error {
			if err := uc.payouts.Save(txCtx, payout); err != nil {
				return err
			}
			return uc.attempts.Save(txCtx, attempt)
		}); persistErr != nil {
			return nil, fmt.Errorf("persist ambiguous payout attempt: %w (provider error: %v)", persistErr, err)
		}
		return nil, fmt.Errorf("initiate provider payout; attempt requires reconciliation: %w", err)
	}
	if err := validatePayoutInitiationResult(result); err != nil {
		return nil, err
	}
	if result.ProviderReference != "" {
		if err := attempt.SetProviderReference(result.ProviderReference); err != nil {
			return nil, fmt.Errorf("set provider payout reference: %w", err)
		}
	}

	if err := applyPayoutInitiationStatus(payout, attempt, result.Status); err != nil {
		return nil, err
	}

	if err := uc.uow.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.payouts.Save(txCtx, payout); err != nil {
			return fmt.Errorf("save payout result: %w", err)
		}
		if err := uc.attempts.Save(txCtx, attempt); err != nil {
			return fmt.Errorf("save payout attempt result: %w", err)
		}
		if err := enqueuePayoutStatusChanged(txCtx, uc.outbox, payout); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("persist payout provider result: %w", err)
	}

	return payoutOutput(payout, attempt), nil
}

func (uc *InitiatePayoutUseCase) reconcileExisting(
	ctx context.Context,
	payout *entities.Payout,
) (*InitiatePayoutOutput, error) {
	if payout.AttemptID() == "" {
		return nil, fmt.Errorf("existing payout has no current attempt")
	}
	attempt, err := uc.attempts.GetByID(ctx, payout.AttemptID())
	if err != nil {
		return nil, fmt.Errorf("load current payout attempt: %w", err)
	}

	if payout.Status() == enums.PayoutStatusSucceeded {
		return payoutOutput(payout, attempt), nil
	}

	// A confirmed failure must not silently trigger another transfer.
	// Order must explicitly call the retry operation.
	if attempt.Status() == enums.PayoutAttemptStatusFailed {
		return payoutOutput(payout, attempt), nil
	}

	result, err := uc.provider.ReconcilePayout(ctx, ports.PayoutReconciliationRequest{
		Payout:  payout,
		Attempt: attempt,
	})
	if err != nil {
		return nil, fmt.Errorf("reconcile existing payout: %w", err)
	}
	if err := validatePayoutReconciliationResult(payout, attempt, result); err != nil {
		return nil, err
	}

	if result.ProviderReference != "" && attempt.ProviderReference() == "" {
		if err := attempt.SetProviderReference(result.ProviderReference); err != nil {
			return nil, fmt.Errorf("set reconciled provider reference: %w", err)
		}
	}

	if err := applyPayoutReconciliation(payout, attempt, result); err != nil {
		return nil, err
	}

	if err := uc.uow.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.payouts.Save(txCtx, payout); err != nil {
			return fmt.Errorf("save reconciled payout: %w", err)
		}
		if err := uc.attempts.Save(txCtx, attempt); err != nil {
			return fmt.Errorf("save reconciled payout attempt: %w", err)
		}
		if err := enqueuePayoutStatusChanged(txCtx, uc.outbox, payout); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("persist payout reconciliation: %w", err)
	}

	return payoutOutput(payout, attempt), nil
}

func samePayoutRequest(p *entities.Payout, input ports.PayoutRequest) bool {
	return p.BusinessReference() == input.BusinessReference &&
		p.StoreID() == strings.TrimSpace(input.StoreID) &&
		p.OrderID() == strings.TrimSpace(input.OrderID) &&
		p.Reason() == strings.TrimSpace(input.Reason) &&
		p.Amount().Amount() == input.Amount.Amount() &&
		p.Amount().Currency() == input.Amount.Currency() &&
		p.Beneficiary() == normalizedBeneficiary(input.Beneficiary)
}

func normalizedBeneficiary(b entities.PayoutBeneficiary) entities.PayoutBeneficiary {
	b.AccountName = strings.TrimSpace(b.AccountName)
	b.AccountNumber = strings.TrimSpace(b.AccountNumber)
	b.BankCode = strings.TrimSpace(b.BankCode)
	b.BankName = strings.TrimSpace(b.BankName)
	b.CountryCode = strings.ToUpper(strings.TrimSpace(b.CountryCode))
	return b
}

func validatePayoutInitiationResult(r ports.PayoutInitiationResult) error {
	switch r.Status {
	case enums.PayoutAttemptStatusPending,
		enums.PayoutAttemptStatusProcessing,
		enums.PayoutAttemptStatusSucceeded,
		enums.PayoutAttemptStatusRequiresReconciliation:
	default:
		return application.ErrInvalidProviderResponse
	}
	if len(r.Metadata) > 0 && !json.Valid(r.Metadata) {
		return application.ErrInvalidProviderResponse
	}
	return nil
}

func validatePayoutReconciliationResult(
	payout *entities.Payout,
	attempt *entities.PayoutAttempt,
	r ports.PayoutReconciliationResult,
) error {
	switch r.Status {
	case enums.PayoutAttemptStatusPending,
		enums.PayoutAttemptStatusProcessing,
		enums.PayoutAttemptStatusSucceeded,
		enums.PayoutAttemptStatusFailed,
		enums.PayoutAttemptStatusRequiresReconciliation:
	default:
		return application.ErrInvalidProviderResponse
	}
	if r.Amount != payout.Amount().Amount() || r.Currency != payout.Amount().Currency() {
		return application.ErrInvalidProviderResponse
	}
	if attempt.ProviderReference() != "" &&
		r.ProviderReference != attempt.ProviderReference() {
		return application.ErrInvalidProviderResponse
	}
	if r.Status == enums.PayoutAttemptStatusFailed &&
		strings.TrimSpace(r.FailureReason) == "" {
		return application.ErrInvalidProviderResponse
	}
	if len(r.Metadata) > 0 && !json.Valid(r.Metadata) {
		return application.ErrInvalidProviderResponse
	}
	return nil
}

func applyPayoutInitiationStatus(
	payout *entities.Payout,
	attempt *entities.PayoutAttempt,
	status enums.PayoutAttemptStatus,
) error {
	if status != enums.PayoutAttemptStatusPending {
		if err := attempt.MarkProcessing(); err != nil {
			return fmt.Errorf("mark payout attempt processing: %w", err)
		}
		switch status {
		case enums.PayoutAttemptStatusSucceeded:
			if err := attempt.MarkSucceeded(); err != nil {
				return fmt.Errorf("mark payout attempt succeeded: %w", err)
			}
		case enums.PayoutAttemptStatusRequiresReconciliation:
			if err := attempt.MarkRequiresReconciliation(); err != nil {
				return fmt.Errorf("mark payout attempt requires reconciliation: %w", err)
			}
		}
	}
	if err := payout.MarkProcessing(); err != nil {
		return fmt.Errorf("mark payout processing: %w", err)
	}
	if status == enums.PayoutAttemptStatusSucceeded {
		if err := payout.MarkSucceeded(); err != nil {
			return fmt.Errorf("mark payout succeeded: %w", err)
		}
	}
	return nil
}

func applyPayoutReconciliation(
	payout *entities.Payout,
	attempt *entities.PayoutAttempt,
	result ports.PayoutReconciliationResult,
) error {
	switch result.Status {
	case enums.PayoutAttemptStatusPending:
		// Keep the current state. A pending provider result is not a failure.
	case enums.PayoutAttemptStatusProcessing:
		if attempt.Status() == enums.PayoutAttemptStatusPending {
			if err := attempt.MarkProcessing(); err != nil {
				return err
			}
		}
	case enums.PayoutAttemptStatusRequiresReconciliation:
		if attempt.Status() == enums.PayoutAttemptStatusProcessing {
			if err := attempt.MarkRequiresReconciliation(); err != nil {
				return err
			}
		}
	case enums.PayoutAttemptStatusSucceeded:
		if attempt.Status() == enums.PayoutAttemptStatusPending {
			if err := attempt.MarkProcessing(); err != nil {
				return err
			}
		}
		if err := attempt.MarkSucceeded(); err != nil {
			return err
		}
	case enums.PayoutAttemptStatusFailed:
		if attempt.Status() == enums.PayoutAttemptStatusPending {
			if err := attempt.MarkProcessing(); err != nil {
				return err
			}
		}
		if err := attempt.MarkFailed(result.FailureReason); err != nil {
			return err
		}
	}

	if payout.Status() == enums.PayoutStatusPending ||
		payout.Status() == enums.PayoutStatusFailed {
		if result.Status != enums.PayoutAttemptStatusFailed {
			if err := payout.MarkProcessing(); err != nil {
				return err
			}
		}
	}
	switch result.Status {
	case enums.PayoutAttemptStatusSucceeded:
		if payout.Status() == enums.PayoutStatusProcessing {
			return payout.MarkSucceeded()
		}
	case enums.PayoutAttemptStatusFailed:
		if payout.Status() == enums.PayoutStatusProcessing {
			return payout.MarkFailed()
		}
	}
	return nil
}

func payoutOutput(
	payout *entities.Payout,
	attempt *entities.PayoutAttempt,
) *InitiatePayoutOutput {
	return &InitiatePayoutOutput{
		PayoutID:      payout.ID(),
		AttemptID:     attempt.ID(),
		Reference:     payout.Reference(),
		Status:        payout.Status(),
		AttemptStatus: attempt.Status(),
		Provider:      attempt.Provider(),
		Amount:        payout.Amount().Amount(),
		Currency:      payout.Amount().Currency(),
	}
}
