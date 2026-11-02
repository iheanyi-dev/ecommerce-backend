package infrastructure

import (
	"fmt"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/use_cases"
)

const paymentProviderName = "paystack"

// UseCases contains the Payment service's application operations.
type UseCases struct {
	InitiatePayment *use_cases.InitiatePaymentUseCase
	VerifyPayment   *use_cases.VerifyPaymentUseCase
	GetPayment      *use_cases.GetPaymentByIDUseCase
	ListPayments    *use_cases.ListPaymentsByPayerIDUseCase

	InitiatePayout *use_cases.InitiatePayoutUseCase
	VerifyPayout   *use_cases.VerifyPayoutUseCase
	GetPayout      *use_cases.GetPayoutUseCase
	ListPayouts    *use_cases.ListPayoutsByStoreIDUseCase
	RetryPayout    *use_cases.RetryPayoutUseCase

	HandleWebhook *use_cases.HandleProviderWebhookUseCase
}

// NewUseCases constructs all Payment application use cases.
func NewUseCases(
	payments ports.PaymentRepository,
	paymentAttempts ports.PaymentAttemptRepository,
	payouts ports.PayoutRepository,
	payoutAttempts ports.PayoutAttemptRepository,
	uow ports.UnitOfWork,
	paymentProvider ports.PaymentProvider,
	payoutProvider ports.PayoutProvider,
	webhookProvider ports.WebhookProvider,
) (*UseCases, error) {
	initiatePayment, err := use_cases.NewInitiatePaymentUseCase(
		payments, paymentAttempts, paymentProvider, uow, paymentProviderName,
	)
	if err != nil {
		return nil, fmt.Errorf("construct initiate payment use case: %w", err)
	}

	verifyPayment, err := use_cases.NewVerifyPaymentUseCase(
		payments, paymentAttempts, paymentProvider, uow,
	)
	if err != nil {
		return nil, fmt.Errorf("construct verify payment use case: %w", err)
	}

	getPayment, err := use_cases.NewGetPaymentByIDUseCase(payments, paymentAttempts)
	if err != nil {
		return nil, fmt.Errorf("construct get payment use case: %w", err)
	}

	listPayments, err := use_cases.NewListPaymentsByPayerIDUseCase(
		payments, paymentAttempts,
	)
	if err != nil {
		return nil, fmt.Errorf("construct list payments use case: %w", err)
	}

	initiatePayout, err := use_cases.NewInitiatePayoutUseCase(
		payments, payouts, payoutAttempts, payoutProvider, uow, paymentProviderName,
	)
	if err != nil {
		return nil, fmt.Errorf("construct initiate payout use case: %w", err)
	}

	verifyPayout, err := use_cases.NewVerifyPayoutUseCase(
		payouts, payoutAttempts, payoutProvider, uow,
	)
	if err != nil {
		return nil, fmt.Errorf("construct verify payout use case: %w", err)
	}

	getPayout, err := use_cases.NewGetPayoutUseCase(payouts)
	if err != nil {
		return nil, fmt.Errorf("construct get payout use case: %w", err)
	}

	listPayouts, err := use_cases.NewListPayoutsByStoreIDUseCase(payouts)
	if err != nil {
		return nil, fmt.Errorf("construct list payouts use case: %w", err)
	}

	retryPayout, err := use_cases.NewRetryPayoutUseCase(
		payouts, payoutAttempts, payoutProvider, uow, paymentProviderName,
	)
	if err != nil {
		return nil, fmt.Errorf("construct retry payout use case: %w", err)
	}

	handleWebhook, err := use_cases.NewHandleProviderWebhookUseCase(
		webhookProvider, verifyPayment, verifyPayout,
	)
	if err != nil {
		return nil, fmt.Errorf("construct provider webhook use case: %w", err)
	}

	return &UseCases{
		InitiatePayment: initiatePayment,
		VerifyPayment:   verifyPayment,
		GetPayment:      getPayment,
		ListPayments:    listPayments,
		InitiatePayout:  initiatePayout,
		VerifyPayout:    verifyPayout,
		GetPayout:       getPayout,
		ListPayouts:     listPayouts,
		RetryPayout:     retryPayout,
		HandleWebhook:   handleWebhook,
	}, nil
}
