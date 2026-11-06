package use_cases

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
)

// VerifyPaymentUseCase reconciles a payment with the authoritative provider
// and persists any confirmed state transition.
type VerifyPaymentUseCase struct {
	payments ports.PaymentRepository
	attempts ports.PaymentAttemptRepository
	outbox   ports.PaymentOutboxRepository
	provider ports.PaymentProvider
	uow      ports.UnitOfWork
}

// NewVerifyPaymentUseCase constructs the payment verification use case.
func NewVerifyPaymentUseCase(
	payments ports.PaymentRepository,
	attempts ports.PaymentAttemptRepository,
	outbox ports.PaymentOutboxRepository,
	provider ports.PaymentProvider,
	uow ports.UnitOfWork,
) (*VerifyPaymentUseCase, error) {
	if payments == nil || attempts == nil || outbox == nil || provider == nil || uow == nil {
		return nil, fmt.Errorf("payment verification dependencies are required")
	}

	return &VerifyPaymentUseCase{
		payments: payments,
		attempts: attempts,
		outbox:   outbox,
		provider: provider,
		uow:      uow,
	}, nil
}

// Execute verifies the provider's current outcome for a payment.
// Provider calls happen outside the database transaction; confirmed changes
// to the payment and attempt are persisted together in a short transaction.
func (uc *VerifyPaymentUseCase) Execute(
	ctx context.Context,
	paymentID string,
) (*PaymentDetails, error) {
	if err := application.ValidateUUID(paymentID); err != nil {
		return nil, fmt.Errorf("payment ID: %w", err)
	}

	payment, err := uc.payments.GetByID(ctx, paymentID)
	if err != nil {
		return nil, fmt.Errorf("get payment for verification: %w", err)
	}
	if payment == nil {
		return nil, application.ErrPaymentNotFound
	}
	if payment.AttemptID() == "" {
		return nil, application.ErrPaymentAttemptNotFound
	}

	attempt, err := uc.attempts.GetByID(ctx, payment.AttemptID())
	if err != nil {
		return nil, fmt.Errorf("%w: %v", application.ErrPaymentAttemptNotFound, err)
	}
	if attempt == nil || attempt.PaymentID() != payment.ID() {
		return nil, application.ErrInvalidProviderResponse
	}

	if payment.Status() == enums.PaymentStatusSucceeded ||
		payment.Status() == enums.PaymentStatusFailed {
		if (payment.Status() == enums.PaymentStatusSucceeded &&
			attempt.Status() != enums.PaymentAttemptStatusSucceeded) ||
			(payment.Status() == enums.PaymentStatusFailed &&
				attempt.Status() != enums.PaymentAttemptStatusFailed) {
			return nil, application.ErrInvalidProviderResponse
		}
		return loadPaymentDetails(ctx, payment, uc.attempts)
	}

	result, err := uc.provider.ReconcileAttempt(ctx, ports.ReconciliationRequest{
		Payment: payment,
		Attempt: attempt,
	})
	if err != nil {
		return nil, fmt.Errorf("reconcile payment with provider: %w", err)
	}
	if err := validateVerificationResult(payment, attempt, result); err != nil {
		return nil, err
	}

	err = uc.uow.WithinTransaction(ctx, func(txCtx context.Context) error {
		// Reload both entities inside the transaction so stale snapshots from
		// the provider call cannot blindly overwrite a newer payment outcome.
		currentPayment, err := uc.payments.GetByID(txCtx, paymentID)
		if err != nil {
			return fmt.Errorf("reload payment in transaction: %w", err)
		}
		if currentPayment == nil || currentPayment.AttemptID() == "" {
			return application.ErrPaymentNotFound
		}

		currentAttempt, err := uc.attempts.GetByID(txCtx, currentPayment.AttemptID())
		if err != nil {
			return fmt.Errorf("reload attempt in transaction: %w", err)
		}
		if currentAttempt == nil || currentAttempt.PaymentID() != currentPayment.ID() ||
			currentAttempt.ID() != attempt.ID() {
			return application.ErrInvalidProviderResponse
		}

		// Another verification may have completed while the provider was
		// being queried. Accept only an identical terminal outcome.
		if currentPayment.Status() == enums.PaymentStatusSucceeded ||
			currentPayment.Status() == enums.PaymentStatusFailed {
			if (result.Status == enums.PaymentAttemptStatusSucceeded &&
				currentPayment.Status() == enums.PaymentStatusSucceeded &&
				currentAttempt.Status() == enums.PaymentAttemptStatusSucceeded) ||
				(result.Status == enums.PaymentAttemptStatusFailed &&
					currentPayment.Status() == enums.PaymentStatusFailed &&
					currentAttempt.Status() == enums.PaymentAttemptStatusFailed &&
					currentAttempt.FailureReason() == strings.TrimSpace(result.FailureReason)) {
				return nil
			}
			return application.ErrInvalidProviderResponse
		}

		if currentAttempt.Status() == enums.PaymentAttemptStatusSucceeded ||
			currentAttempt.Status() == enums.PaymentAttemptStatusFailed {
			return application.ErrInvalidProviderResponse
		}

		switch result.Status {
		case enums.PaymentAttemptStatusPending,
			enums.PaymentAttemptStatusProcessing:
			if currentAttempt.Status() == enums.PaymentAttemptStatusRequiresReconciliation {
				return nil
			}
			if currentAttempt.Status() == enums.PaymentAttemptStatusPending &&
				result.Status == enums.PaymentAttemptStatusProcessing {
				if err := currentAttempt.MarkProcessing(); err != nil {
					return fmt.Errorf("mark attempt processing: %w", err)
				}
				if currentPayment.Status() == enums.PaymentStatusPending {
					if err := currentPayment.MarkProcessing(); err != nil {
						return fmt.Errorf("mark payment processing: %w", err)
					}
				}
				if err := uc.payments.Save(txCtx, currentPayment); err != nil {
					return fmt.Errorf("save processing payment: %w", err)
				}
				if err := uc.attempts.Save(txCtx, currentAttempt); err != nil {
					return fmt.Errorf("save processing attempt: %w", err)
				}
			} else if currentAttempt.Status() != enums.PaymentAttemptStatusPending &&
				currentAttempt.Status() != enums.PaymentAttemptStatusProcessing {
				return application.ErrInvalidProviderResponse
			}
			return nil

		case enums.PaymentAttemptStatusRequiresReconciliation:
			if currentAttempt.Status() == enums.PaymentAttemptStatusPending {
				if err := currentAttempt.MarkProcessing(); err != nil {
					return fmt.Errorf("mark attempt processing: %w", err)
				}
			}
			if currentAttempt.Status() == enums.PaymentAttemptStatusProcessing {
				if err := currentAttempt.MarkRequiresReconciliation(); err != nil {
					return fmt.Errorf("mark attempt for reconciliation: %w", err)
				}
				if currentPayment.Status() == enums.PaymentStatusPending {
					if err := currentPayment.MarkProcessing(); err != nil {
						return fmt.Errorf("mark payment processing: %w", err)
					}
				}
				if err := uc.payments.Save(txCtx, currentPayment); err != nil {
					return fmt.Errorf("save reconciliation payment: %w", err)
				}
				if err := uc.attempts.Save(txCtx, currentAttempt); err != nil {
					return fmt.Errorf("save reconciliation attempt: %w", err)
				}
				return nil
			}
			if currentAttempt.Status() == enums.PaymentAttemptStatusRequiresReconciliation {
				return nil
			}
			return application.ErrInvalidProviderResponse

		case enums.PaymentAttemptStatusSucceeded:
			if currentAttempt.Status() == enums.PaymentAttemptStatusPending {
				if err := currentAttempt.MarkProcessing(); err != nil {
					return fmt.Errorf("mark attempt processing: %w", err)
				}
			}
			if err := currentAttempt.MarkSucceeded(); err != nil {
				return fmt.Errorf("mark attempt succeeded: %w", err)
			}
			if currentPayment.Status() == enums.PaymentStatusPending {
				if err := currentPayment.MarkProcessing(); err != nil {
					return fmt.Errorf("mark payment processing: %w", err)
				}
			}
			if err := currentPayment.MarkSucceeded(); err != nil {
				return fmt.Errorf("mark payment succeeded: %w", err)
			}

		case enums.PaymentAttemptStatusFailed:
			if currentAttempt.Status() == enums.PaymentAttemptStatusPending {
				if err := currentAttempt.MarkProcessing(); err != nil {
					return fmt.Errorf("mark attempt processing: %w", err)
				}
			}
			if err := currentAttempt.MarkFailed(result.FailureReason); err != nil {
				return fmt.Errorf("mark attempt failed: %w", err)
			}
			if currentPayment.Status() == enums.PaymentStatusPending {
				if err := currentPayment.MarkProcessing(); err != nil {
					return fmt.Errorf("mark payment processing: %w", err)
				}
			}
			if err := currentPayment.MarkFailed(); err != nil {
				return fmt.Errorf("mark payment failed: %w", err)
			}

		default:
			return application.ErrInvalidProviderResponse
		}

		if err := uc.payments.Save(txCtx, currentPayment); err != nil {
			return fmt.Errorf("save verified payment: %w", err)
		}
		if err := uc.attempts.Save(txCtx, currentAttempt); err != nil {
			return fmt.Errorf("save verified attempt: %w", err)
		}
		if err := enqueuePaymentStatusChanged(txCtx, uc.outbox, currentPayment); err != nil {
			return fmt.Errorf("enqueue payment status event: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("persist verification result: %w", err)
	}

	payment, err = uc.payments.GetByID(ctx, paymentID)
	if err != nil {
		return nil, fmt.Errorf("reload verified payment: %w", err)
	}
	if payment == nil {
		return nil, application.ErrPaymentNotFound
	}
	return loadPaymentDetails(ctx, payment, uc.attempts)
}

// validateVerificationResult rejects provider outcomes that do not match the
// stored payment, including mismatched transaction identity or money values.
func validateVerificationResult(
	payment *entities.Payment,
	attempt *entities.PaymentAttempt,
	result ports.ReconciliationResult,
) error {
	if !result.Status.IsValid() {
		return application.ErrInvalidProviderResponse
	}
	if result.Amount <= 0 ||
		result.Amount != payment.Amount().Amount() ||
		!strings.EqualFold(strings.TrimSpace(result.Currency), payment.Amount().Currency()) {
		return application.ErrInvalidProviderResponse
	}
	expectedReference := attempt.ProviderReference()
	if expectedReference == "" {
		expectedReference = attempt.ID()
	}
	if strings.TrimSpace(result.ProviderReference) == "" ||
		result.ProviderReference != expectedReference {
		return application.ErrInvalidProviderResponse
	}
	if result.Status == enums.PaymentAttemptStatusFailed &&
		strings.TrimSpace(result.FailureReason) == "" {
		return application.ErrInvalidProviderResponse
	}
	if len(result.Metadata) > 0 && !json.Valid(result.Metadata) {
		return application.ErrInvalidProviderResponse
	}
	return nil
}

// ensureAttemptCanFinalize rejects contradictory results for terminal attempts.
func ensureAttemptCanFinalize(
	attempt *entities.PaymentAttempt,
	success bool,
	reason string,
) error {
	switch attempt.Status() {
	case enums.PaymentAttemptStatusProcessing,
		enums.PaymentAttemptStatusRequiresReconciliation,
		enums.PaymentAttemptStatusPending:
		return nil
	case enums.PaymentAttemptStatusSucceeded,
		enums.PaymentAttemptStatusFailed:
		// The payment is still nonterminal when this helper is called.
		// A terminal attempt in that situation is an inconsistent state.
		return application.ErrInvalidProviderResponse
	}
	return application.ErrInvalidProviderResponse
}
