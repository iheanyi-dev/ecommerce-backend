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

type RetryPayoutUseCase struct {
	payouts      ports.PayoutRepository
	attempts     ports.PayoutAttemptRepository
	provider     ports.PayoutProvider
	uow          ports.UnitOfWork
	providerName string
}

func NewRetryPayoutUseCase(
	payouts ports.PayoutRepository,
	attempts ports.PayoutAttemptRepository,
	provider ports.PayoutProvider,
	uow ports.UnitOfWork,
	providerName string,
) (*RetryPayoutUseCase, error) {
	if payouts == nil || attempts == nil || provider == nil || uow == nil {
		return nil, fmt.Errorf("payout retry dependencies are required")
	}
	if strings.TrimSpace(providerName) == "" {
		return nil, fmt.Errorf("provider name is required")
	}
	return &RetryPayoutUseCase{
		payouts:      payouts,
		attempts:     attempts,
		provider:     provider,
		uow:          uow,
		providerName: strings.TrimSpace(providerName),
	}, nil
}

// Execute is called only when the Order service explicitly requests a retry.
// It never reconciles an ambiguous attempt by starting another transfer.
func (uc *RetryPayoutUseCase) Execute(
	ctx context.Context,
	payoutID string,
	storeID string,
) (*InitiatePayoutOutput, error) {
	payoutID = strings.TrimSpace(payoutID)
	storeID = strings.TrimSpace(storeID)
	if payoutID == "" || storeID == "" {
		return nil, fmt.Errorf("payout ID and store ID are required")
	}

	payout, err := uc.payouts.GetByID(ctx, payoutID)
	if err != nil {
		if errors.Is(err, application.ErrPayoutNotFound) {
			return nil, application.ErrPayoutNotFound
		}
		return nil, fmt.Errorf("load payout for retry: %w", err)
	}
	if payout.StoreID() != storeID {
		return nil, application.ErrPayoutNotFound
	}
	if payout.Status() != enums.PayoutStatusFailed || payout.AttemptID() == "" {
		return nil, application.ErrPayoutRetryNotAllowed
	}

	previous, err := uc.attempts.GetByID(ctx, payout.AttemptID())
	if err != nil {
		return nil, fmt.Errorf("load previous payout attempt: %w", err)
	}
	if previous.Status() != enums.PayoutAttemptStatusFailed {
		return nil, application.ErrPayoutRetryNotAllowed
	}

	attempt, err := entities.NewPayoutAttempt(payout.ID(), uc.providerName)
	if err != nil {
		return nil, fmt.Errorf("create retry attempt: %w", err)
	}
	if err := payout.SetAttemptID(attempt.ID()); err != nil {
		return nil, fmt.Errorf("link retry attempt: %w", err)
	}
	if err := payout.MarkProcessing(); err != nil {
		return nil, fmt.Errorf("mark payout processing: %w", err)
	}

	// Persist the new attempt before calling the provider. A crash after this
	// point can be reconciled without creating a second transfer.
	if err := uc.uow.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.payouts.Save(txCtx, payout); err != nil {
			return fmt.Errorf("save payout before retry: %w", err)
		}
		if err := uc.attempts.Save(txCtx, attempt); err != nil {
			return fmt.Errorf("save retry attempt: %w", err)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("persist retry attempt: %w", err)
	}

	result, providerErr := uc.provider.InitiatePayout(ctx, ports.PayoutInitiationRequest{
		Payout:  payout,
		Attempt: attempt,
	})
	if providerErr != nil {
		// A network error does not prove the transfer failed. Require
		// reconciliation; never mark this attempt definitively failed.
		if err := attempt.MarkProcessing(); err != nil {
			return nil, fmt.Errorf("mark ambiguous retry processing: %w", err)
		}
		if err := attempt.MarkRequiresReconciliation(); err != nil {
			return nil, fmt.Errorf("mark retry for reconciliation: %w", err)
		}
		if err := uc.saveRetryState(ctx, payout, attempt); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("provider retry outcome requires reconciliation: %w", providerErr)
	}

	if err := validatePayoutInitiationResult(result); err != nil {
		// The provider may already have initiated the transfer. Keep the
		// attempt reconcilable rather than permitting an unsafe retry.
		if attempt.Status() == enums.PayoutAttemptStatusPending {
			if transitionErr := attempt.MarkProcessing(); transitionErr != nil {
				return nil, transitionErr
			}
		}
		if attempt.Status() == enums.PayoutAttemptStatusProcessing {
			if transitionErr := attempt.MarkRequiresReconciliation(); transitionErr != nil {
				return nil, transitionErr
			}
		}
		if saveErr := uc.saveRetryState(ctx, payout, attempt); saveErr != nil {
			return nil, saveErr
		}
		return nil, err
	}

	if result.ProviderReference != "" {
		if err := attempt.SetProviderReference(result.ProviderReference); err != nil {
			return nil, fmt.Errorf("set retry provider reference: %w", err)
		}
	}

	switch result.Status {
	case enums.PayoutAttemptStatusPending:
		// Leave the attempt pending. The payout remains processing.
	case enums.PayoutAttemptStatusProcessing:
		if err := attempt.MarkProcessing(); err != nil {
			return nil, fmt.Errorf("mark retry processing: %w", err)
		}
	case enums.PayoutAttemptStatusSucceeded:
		if err := attempt.MarkProcessing(); err != nil {
			return nil, fmt.Errorf("mark retry processing: %w", err)
		}
		if err := attempt.MarkSucceeded(); err != nil {
			return nil, fmt.Errorf("mark retry succeeded: %w", err)
		}
		if err := payout.MarkSucceeded(); err != nil {
			return nil, fmt.Errorf("mark payout succeeded: %w", err)
		}
	case enums.PayoutAttemptStatusRequiresReconciliation:
		if err := attempt.MarkProcessing(); err != nil {
			return nil, fmt.Errorf("mark retry processing: %w", err)
		}
		if err := attempt.MarkRequiresReconciliation(); err != nil {
			return nil, fmt.Errorf("mark retry requires reconciliation: %w", err)
		}
	default:
		return nil, application.ErrInvalidProviderResponse
	}

	if len(result.Metadata) > 0 && !json.Valid(result.Metadata) {
		return nil, application.ErrInvalidProviderResponse
	}
	if err := uc.saveRetryState(ctx, payout, attempt); err != nil {
		return nil, err
	}
	return payoutOutput(payout, attempt), nil
}

func (uc *RetryPayoutUseCase) saveRetryState(
	ctx context.Context,
	payout *entities.Payout,
	attempt *entities.PayoutAttempt,
) error {
	if err := uc.uow.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.payouts.Save(txCtx, payout); err != nil {
			return fmt.Errorf("save retried payout: %w", err)
		}
		if err := uc.attempts.Save(txCtx, attempt); err != nil {
			return fmt.Errorf("save retried payout attempt: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("persist retry outcome: %w", err)
	}
	return nil
}
