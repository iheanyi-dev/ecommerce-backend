package use_cases

import (
	"context"
	"encoding/json"
	"fmt"
	"net/mail"
	"net/url"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/value_objects"
)

// InitiatePaymentInput contains the information required to initiate a payment.
// All IDs must be UUIDs. The business reference identifies the operation being paid for.
type InitiatePaymentInput struct {
	PayerID         string
	Purpose         enums.PaymentPurpose
	BusinessRefType string
	BusinessRefID   string
	Amount          value_objects.Money
	CustomerEmail   string
}

// InitiatePaymentOutput contains the payment and checkout details returned to the caller.
type InitiatePaymentOutput struct {
	PaymentID     string
	AttemptID     string
	PaymentStatus enums.PaymentStatus
	AttemptStatus enums.PaymentAttemptStatus
	Provider      string
	CheckoutURL   string
	Amount        int64
	Currency      string
}

// InitiatePaymentUseCase coordinates idempotent payment creation and provider checkout.
type InitiatePaymentUseCase struct {
	payments     ports.PaymentRepository
	attempts     ports.PaymentAttemptRepository
	provider     ports.PaymentProvider
	uow          ports.UnitOfWork
	providerName string
}

// NewInitiatePaymentUseCase creates a payment initiation use case.
func NewInitiatePaymentUseCase(
	payments ports.PaymentRepository,
	attempts ports.PaymentAttemptRepository,
	provider ports.PaymentProvider,
	uow ports.UnitOfWork,
	providerName string,
) (*InitiatePaymentUseCase, error) {
	if payments == nil || attempts == nil || provider == nil || uow == nil {
		return nil, fmt.Errorf("payment initiation dependencies are required")
	}
	if strings.TrimSpace(providerName) == "" {
		return nil, fmt.Errorf("provider name is required")
	}
	return &InitiatePaymentUseCase{
		payments:     payments,
		attempts:     attempts,
		provider:     provider,
		uow:          uow,
		providerName: providerName,
	}, nil
}

func (uc *InitiatePaymentUseCase) Execute(
	ctx context.Context,
	input InitiatePaymentInput,
) (*InitiatePaymentOutput, error) {
	if err := application.ValidateUUID(input.PayerID); err != nil {
		return nil, fmt.Errorf("payer ID: %w", err)
	}
	if err := application.ValidateUUID(input.BusinessRefID); err != nil {
		return nil, fmt.Errorf("business reference ID: %w", err)
	}
	if strings.TrimSpace(input.BusinessRefType) == "" {
		return nil, fmt.Errorf("business reference type is required")
	}
	input.CustomerEmail = strings.TrimSpace(input.CustomerEmail)
	emailAddress, emailErr := mail.ParseAddress(input.CustomerEmail)
	if emailErr != nil || emailAddress.Address != input.CustomerEmail {
		return nil, fmt.Errorf("valid customer email is required")
	}
	if input.Amount.IsZero() {
		return nil, fmt.Errorf("payment amount must be positive and use a supported currency")
	}

	ref, err := value_objects.NewBusinessReference(input.BusinessRefType, input.BusinessRefID)
	if err != nil {
		return nil, fmt.Errorf("business reference: %w", err)
	}

	payment, err := uc.payments.GetByBusinessReference(ctx, ref)
	if err != nil {
		if err != application.ErrPaymentNotFound {
			return nil, fmt.Errorf("find payment by business reference: %w", err)
		}

		payment, err = entities.NewPayment(input.PayerID, input.Purpose, ref, input.Amount)
		if err != nil {
			return nil, fmt.Errorf("create payment: %w", err)
		}
	} else {
		if payment.PayerID() != input.PayerID ||
			payment.Purpose() != input.Purpose ||
			payment.Amount().Amount() != input.Amount.Amount() ||
			payment.Amount().Currency() != input.Amount.Currency() {
			return nil, application.ErrIdempotencyConflict
		}
	}

	// A previously created attempt must be reconciled before any replacement attempt.
	// The domain entity is expected to expose AttemptID() and SetAttemptID().
	attemptID := payment.AttemptID()
	if attemptID != "" {
		attempt, getErr := uc.attempts.GetByID(ctx, attemptID)
		if getErr != nil {
			return nil, fmt.Errorf("load current payment attempt: %w", getErr)
		}

		reconciled, reconcileErr := uc.provider.ReconcileAttempt(ctx, ports.ReconciliationRequest{
			Payment: payment,
			Attempt: attempt,
		})
		if reconcileErr != nil {
			return nil, fmt.Errorf("reconcile payment attempt: %w", reconcileErr)
		}

		if err := validateReconciliation(reconciled); err != nil {
			return nil, err
		}

		if reconciled.Status != enums.PaymentAttemptStatusFailed {
			return &InitiatePaymentOutput{
				PaymentID:     payment.ID(),
				AttemptID:     attempt.ID(),
				PaymentStatus: payment.Status(),
				AttemptStatus: reconciled.Status,
				Provider:      attempt.Provider(),
				CheckoutURL:   reconciled.CheckoutURL,
				Amount:        payment.Amount().Amount(),
				Currency:      payment.Amount().Currency(),
			}, nil
		}

		if reconciled.FailureReason == "" {
			return nil, application.ErrInvalidProviderResponse
		}
		// A failed attempt is not overwritten. A replacement attempt is created below.
	}

	attempt, err := entities.NewPaymentAttempt(payment.ID(), uc.providerName)
	if err != nil {
		return nil, fmt.Errorf("create payment attempt: %w", err)
	}

	if err := attempt.SetCustomerEmail(input.CustomerEmail); err != nil {
		return nil, fmt.Errorf("set payment attempt customer email: %w", err)
	}

	// Persist the payment and attempt before contacting the provider.
	// If the provider accepts the request but its response is lost, this
	// durable attempt can be reconciled using its stable attempt UUID.
	if err := payment.SetAttemptID(attempt.ID()); err != nil {
		return nil, fmt.Errorf("link payment attempt: %w", err)
	}
	if err := uc.uow.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.payments.Save(txCtx, payment); err != nil {
			return fmt.Errorf("save payment: %w", err)
		}
		if err := uc.attempts.Save(txCtx, attempt); err != nil {
			return fmt.Errorf("save payment attempt: %w", err)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("persist payment before provider initiation: %w", err)
	}

	result, err := uc.provider.InitiateCheckout(ctx, ports.CheckoutRequest{
		PaymentID:         payment.ID(),
		AttemptID:         attempt.ID(),
		PaymentReference:  payment.Reference(),
		CustomerEmail:     attempt.CustomerEmail(),
		BusinessReference: ref,
		Amount:            payment.Amount(),
	})
	if err != nil {
		// The request may have reached the provider. Never blindly retry it.
		if transitionErr := attempt.MarkProcessing(); transitionErr == nil {
			transitionErr = attempt.MarkRequiresReconciliation()
			if transitionErr != nil {
				return nil, fmt.Errorf("mark payment attempt for reconciliation: %w", transitionErr)
			}
		} else {
			return nil, fmt.Errorf("mark payment attempt processing after provider error: %w", transitionErr)
		}
		if payment.Status() == enums.PaymentStatusPending {
			if transitionErr := payment.MarkProcessing(); transitionErr != nil {
				return nil, fmt.Errorf("mark payment processing after provider error: %w", transitionErr)
			}
		}
		if persistErr := uc.uow.WithinTransaction(ctx, func(txCtx context.Context) error {
			if err := uc.payments.Save(txCtx, payment); err != nil {
				return err
			}
			return uc.attempts.Save(txCtx, attempt)
		}); persistErr != nil {
			return nil, fmt.Errorf("persist ambiguous payment attempt: %w (provider error: %v)", persistErr, err)
		}
		return nil, fmt.Errorf("initiate provider checkout; attempt requires reconciliation: %w", err)
	}
	if err := validateCheckoutResult(result); err != nil {
		return nil, err
	}

	if result.ProviderReference != "" {
		if err := attempt.SetProviderReference(result.ProviderReference); err != nil {
			return nil, fmt.Errorf("set provider reference: %w", err)
		}
	}

	// A newly initiated checkout must have a usable URL for the caller.
	// Preserve provider metadata and expiry on the attempt entity.
	if err := attempt.SetCheckoutDetails(result.CheckoutURL, result.ExpiresAt, result.Metadata); err != nil {
		return nil, fmt.Errorf("set checkout details: %w", err)
	}

	if err := applyAttemptStatus(attempt, result.Status); err != nil {
		return nil, err
	}
	if payment.Status() == enums.PaymentStatusPending {
		if err := payment.MarkProcessing(); err != nil {
			return nil, fmt.Errorf("mark payment processing: %w", err)
		}
	}
	if result.Status == enums.PaymentAttemptStatusSucceeded {
		if err := payment.MarkSucceeded(); err != nil {
			return nil, fmt.Errorf("mark payment succeeded: %w", err)
		}
	}
	if result.Status == enums.PaymentAttemptStatusFailed {
		if err := payment.MarkFailed(); err != nil {
			return nil, fmt.Errorf("mark payment failed: %w", err)
		}
	}

	err = uc.uow.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.payments.Save(txCtx, payment); err != nil {
			return fmt.Errorf("save payment result: %w", err)
		}
		if err := uc.attempts.Save(txCtx, attempt); err != nil {
			return fmt.Errorf("save payment attempt result: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("persist payment provider result: %w", err)
	}

	return &InitiatePaymentOutput{
		PaymentID:     payment.ID(),
		AttemptID:     attempt.ID(),
		PaymentStatus: payment.Status(),
		AttemptStatus: attempt.Status(),
		Provider:      attempt.Provider(),
		CheckoutURL:   result.CheckoutURL,
		Amount:        payment.Amount().Amount(),
		Currency:      payment.Amount().Currency(),
	}, nil
}

func validateCheckoutResult(result ports.CheckoutResult) error {
	if strings.TrimSpace(result.CheckoutURL) == "" {
		return application.ErrInvalidProviderResponse
	}
	parsed, err := url.ParseRequestURI(result.CheckoutURL)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return application.ErrInvalidProviderResponse
	}
	if result.Status != enums.PaymentAttemptStatusPending &&
		result.Status != enums.PaymentAttemptStatusProcessing &&
		result.Status != enums.PaymentAttemptStatusSucceeded &&
		result.Status != enums.PaymentAttemptStatusFailed &&
		result.Status != enums.PaymentAttemptStatusRequiresReconciliation {
		return application.ErrInvalidProviderResponse
	}
	if len(result.Metadata) > 0 && !json.Valid(result.Metadata) {
		return application.ErrInvalidProviderResponse
	}
	if result.Status == enums.PaymentAttemptStatusFailed {
		return application.ErrInvalidProviderResponse
	}
	return nil
}

func validateReconciliation(result ports.ReconciliationResult) error {
	switch result.Status {
	case enums.PaymentAttemptStatusPending,
		enums.PaymentAttemptStatusProcessing,
		enums.PaymentAttemptStatusSucceeded,
		enums.PaymentAttemptStatusFailed,
		enums.PaymentAttemptStatusRequiresReconciliation:
	default:
		return application.ErrInvalidProviderResponse
	}
	if result.Status == enums.PaymentAttemptStatusFailed && strings.TrimSpace(result.FailureReason) == "" {
		return application.ErrInvalidProviderResponse
	}
	if result.CheckoutURL != "" {
		parsed, err := url.ParseRequestURI(result.CheckoutURL)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
			return application.ErrInvalidProviderResponse
		}
	}
	if len(result.Metadata) > 0 && !json.Valid(result.Metadata) {
		return application.ErrInvalidProviderResponse
	}
	return nil
}

func applyAttemptStatus(attempt *entities.PaymentAttempt, status enums.PaymentAttemptStatus) error {
	switch status {
	case enums.PaymentAttemptStatusPending:
		return nil
	case enums.PaymentAttemptStatusProcessing:
		return attempt.MarkProcessing()
	case enums.PaymentAttemptStatusSucceeded:
		if err := attempt.MarkProcessing(); err != nil {
			return err
		}
		return attempt.MarkSucceeded()
	case enums.PaymentAttemptStatusRequiresReconciliation:
		if err := attempt.MarkProcessing(); err != nil {
			return err
		}
		return attempt.MarkRequiresReconciliation()
	default:
		return fmt.Errorf("unsupported initial attempt status %q", status)
	}
}
