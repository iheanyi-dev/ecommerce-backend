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

// VerifyPayout reconciles an existing provider transfer. It never initiates
// another transfer; retrying a failed payout remains an explicit operation.
type VerifyPayoutUseCase struct {
	payouts  ports.PayoutRepository
	attempts ports.PayoutAttemptRepository
	provider ports.PayoutProvider
	outbox   ports.PaymentOutboxRepository
	uow      ports.UnitOfWork
}

func NewVerifyPayoutUseCase(
	payouts ports.PayoutRepository,
	attempts ports.PayoutAttemptRepository,
	provider ports.PayoutProvider,
	outbox ports.PaymentOutboxRepository,
	uow ports.UnitOfWork,
) (*VerifyPayoutUseCase, error) {
	if payouts == nil || attempts == nil || provider == nil || outbox == nil || uow == nil {
		return nil, fmt.Errorf("payout verification dependencies are required")
	}
	return &VerifyPayoutUseCase{
		payouts: payouts, attempts: attempts, provider: provider, outbox: outbox, uow: uow,
	}, nil
}

func (uc *VerifyPayoutUseCase) Execute(
	ctx context.Context,
	payoutID string,
) (*InitiatePayoutOutput, error) {
	if err := application.ValidateUUID(payoutID); err != nil {
		return nil, fmt.Errorf("payout ID: %w", err)
	}

	payout, err := uc.payouts.GetByID(ctx, payoutID)
	if err != nil {
		if errors.Is(err, application.ErrPayoutNotFound) {
			return nil, application.ErrPayoutNotFound
		}
		return nil, fmt.Errorf("get payout for verification: %w", err)
	}
	if payout == nil {
		return nil, application.ErrPayoutNotFound
	}
	if payout.AttemptID() == "" {
		return nil, application.ErrPayoutAttemptNotFound
	}

	attempt, err := uc.attempts.GetByID(ctx, payout.AttemptID())
	if err != nil {
		return nil, fmt.Errorf("%w: %v", application.ErrPayoutAttemptNotFound, err)
	}
	if attempt == nil || attempt.PayoutID() != payout.ID() {
		return nil, application.ErrInvalidProviderResponse
	}

	if payout.Status() == enums.PayoutStatusSucceeded ||
		payout.Status() == enums.PayoutStatusFailed {
		if (payout.Status() == enums.PayoutStatusSucceeded &&
			attempt.Status() != enums.PayoutAttemptStatusSucceeded) ||
			(payout.Status() == enums.PayoutStatusFailed &&
				attempt.Status() != enums.PayoutAttemptStatusFailed) {
			return nil, application.ErrInvalidProviderResponse
		}
		return payoutOutput(payout, attempt), nil
	}

	result, err := uc.provider.ReconcilePayout(ctx, ports.PayoutReconciliationRequest{
		Payout:  payout,
		Attempt: attempt,
	})
	if err != nil {
		return nil, fmt.Errorf("reconcile payout with provider: %w", err)
	}
	if err := validatePayoutVerificationResult(payout, attempt, result); err != nil {
		return nil, err
	}

	err = uc.uow.WithinTransaction(ctx, func(txCtx context.Context) error {
		currentPayout, err := uc.payouts.GetByID(txCtx, payoutID)
		if err != nil {
			return fmt.Errorf("reload payout in transaction: %w", err)
		}
		if currentPayout == nil || currentPayout.AttemptID() == "" {
			return application.ErrPayoutNotFound
		}

		currentAttempt, err := uc.attempts.GetByID(txCtx, currentPayout.AttemptID())
		if err != nil {
			return fmt.Errorf("reload payout attempt in transaction: %w", err)
		}
		if currentAttempt == nil ||
			currentAttempt.PayoutID() != currentPayout.ID() ||
			currentAttempt.ID() != attempt.ID() {
			return application.ErrInvalidProviderResponse
		}

		// Another verifier may have completed the same attempt while the
		// provider was being queried. Accept only an identical terminal result.
		if currentAttempt.Status() == enums.PayoutAttemptStatusSucceeded ||
			currentAttempt.Status() == enums.PayoutAttemptStatusFailed {
			if (result.Status == enums.PayoutAttemptStatusSucceeded &&
				currentPayout.Status() == enums.PayoutStatusSucceeded &&
				currentAttempt.Status() == enums.PayoutAttemptStatusSucceeded) ||
				(result.Status == enums.PayoutAttemptStatusFailed &&
					currentPayout.Status() == enums.PayoutStatusFailed &&
					currentAttempt.Status() == enums.PayoutAttemptStatusFailed &&
					currentAttempt.FailureReason() == strings.TrimSpace(result.FailureReason)) {
				return nil
			}
			return application.ErrInvalidProviderResponse
		}

		if currentPayout.Status() == enums.PayoutStatusSucceeded ||
			currentPayout.Status() == enums.PayoutStatusFailed {
			return application.ErrInvalidProviderResponse
		}

		// A failed transfer must pass through processing before the payout
		// aggregate can transition to failed.
		if result.Status == enums.PayoutAttemptStatusFailed &&
			currentPayout.Status() == enums.PayoutStatusPending {
			if err := currentPayout.MarkProcessing(); err != nil {
				return fmt.Errorf("mark payout processing: %w", err)
			}
		}

		if err := applyPayoutReconciliation(currentPayout, currentAttempt, result); err != nil {
			return fmt.Errorf("apply payout verification result: %w", err)
		}

		if err := uc.payouts.Save(txCtx, currentPayout); err != nil {
			return fmt.Errorf("save verified payout: %w", err)
		}
		if err := uc.attempts.Save(txCtx, currentAttempt); err != nil {
			return fmt.Errorf("save verified payout attempt: %w", err)
		}
		if err := enqueuePayoutStatusChanged(txCtx, uc.outbox, currentPayout); err != nil {
			return fmt.Errorf("enqueue payout status event: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("persist payout verification: %w", err)
	}

	payout, err = uc.payouts.GetByID(ctx, payoutID)
	if err != nil {
		return nil, fmt.Errorf("reload verified payout: %w", err)
	}
	if payout == nil || payout.AttemptID() == "" {
		return nil, application.ErrPayoutNotFound
	}
	attempt, err = uc.attempts.GetByID(ctx, payout.AttemptID())
	if err != nil {
		return nil, fmt.Errorf("reload verified payout attempt: %w", err)
	}
	if attempt == nil || attempt.PayoutID() != payout.ID() {
		return nil, application.ErrInvalidProviderResponse
	}
	return payoutOutput(payout, attempt), nil
}

func validatePayoutVerificationResult(
	payout *entities.Payout,
	attempt *entities.PayoutAttempt,
	result ports.PayoutReconciliationResult,
) error {
	switch result.Status {
	case enums.PayoutAttemptStatusPending,
		enums.PayoutAttemptStatusProcessing,
		enums.PayoutAttemptStatusSucceeded,
		enums.PayoutAttemptStatusFailed,
		enums.PayoutAttemptStatusRequiresReconciliation:
	default:
		return application.ErrInvalidProviderResponse
	}

	if result.Amount <= 0 ||
		result.Amount != payout.Amount().Amount() ||
		!strings.EqualFold(strings.TrimSpace(result.Currency), payout.Amount().Currency()) {
		return application.ErrInvalidProviderResponse
	}

	providerReference := strings.TrimSpace(result.ProviderReference)
	expectedReference := attempt.ProviderReference()
	if expectedReference == "" {
		expectedReference = attempt.ID()
	}
	if providerReference == "" || providerReference != expectedReference {
		return application.ErrInvalidProviderResponse
	}

	if result.Status == enums.PayoutAttemptStatusFailed &&
		strings.TrimSpace(result.FailureReason) == "" {
		return application.ErrInvalidProviderResponse
	}
	if len(result.Metadata) > 0 && !json.Valid(result.Metadata) {
		return application.ErrInvalidProviderResponse
	}
	return nil
}
