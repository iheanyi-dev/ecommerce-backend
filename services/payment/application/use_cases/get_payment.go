package use_cases

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/value_objects"
)

// PaymentDetails contains a payment and its current attempt details, if present.
type PaymentDetails struct {
	ID                    string
	Reference             string
	PayerID               string
	Purpose               enums.PaymentPurpose
	BusinessReferenceType string
	BusinessReferenceID   string
	Amount                int64
	Currency              string
	Status                enums.PaymentStatus
	Attempt               *PaymentAttemptDetails
}

// PaymentAttemptDetails contains the provider and checkout information for an attempt.
type PaymentAttemptDetails struct {
	ID                string
	PaymentID         string
	Provider          string
	ProviderReference string
	Status            enums.PaymentAttemptStatus
	FailureReason     string
	CheckoutURL       string
	ExpiresAt         interface{}
	Metadata          json.RawMessage
}

// GetPaymentByIDUseCase retrieves a payment using its UUID.
type GetPaymentByIDUseCase struct {
	payments ports.PaymentRepository
	attempts ports.PaymentAttemptRepository
}

// NewGetPaymentByIDUseCase constructs the payment-by-ID use case.
func NewGetPaymentByIDUseCase(
	payments ports.PaymentRepository,
	attempts ports.PaymentAttemptRepository,
) (*GetPaymentByIDUseCase, error) {
	if payments == nil || attempts == nil {
		return nil, fmt.Errorf("payment and attempt repositories are required")
	}
	return &GetPaymentByIDUseCase{
		payments: payments, attempts: attempts,
	}, nil
}

func (uc *GetPaymentByIDUseCase) Execute(ctx context.Context, paymentID string) (*PaymentDetails, error) {
	if err := application.ValidateUUID(paymentID); err != nil {
		return nil, fmt.Errorf("payment ID: %w", err)
	}

	payment, err := uc.payments.GetByID(ctx, paymentID)
	if err != nil {
		return nil, fmt.Errorf("get payment by ID: %w", err)
	}
	return loadPaymentDetails(ctx, payment, uc.attempts)
}

// GetPaymentByBusinessReferenceUseCase retrieves a payment by its typed business reference.
type GetPaymentByBusinessReferenceUseCase struct {
	payments ports.PaymentRepository
	attempts ports.PaymentAttemptRepository
}

// NewGetPaymentByBusinessReferenceUseCase constructs the business-reference retrieval use case.
func NewGetPaymentByBusinessReferenceUseCase(
	payments ports.PaymentRepository,
	attempts ports.PaymentAttemptRepository,
) (*GetPaymentByBusinessReferenceUseCase, error) {
	if payments == nil || attempts == nil {
		return nil, fmt.Errorf("payment and attempt repositories are required")
	}
	return &GetPaymentByBusinessReferenceUseCase{
		payments: payments, attempts: attempts,
	}, nil
}

func (uc *GetPaymentByBusinessReferenceUseCase) Execute(
	ctx context.Context,
	referenceType string,
	referenceID string,
) (*PaymentDetails, error) {
	if err := application.ValidateUUID(referenceID); err != nil {
		return nil, fmt.Errorf("business reference ID: %w", err)
	}

	ref, err := value_objects.NewBusinessReference(referenceType, referenceID)
	if err != nil {
		return nil, fmt.Errorf("business reference: %w", err)
	}

	payment, err := uc.payments.GetByBusinessReference(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("get payment by business reference: %w", err)
	}
	return loadPaymentDetails(ctx, payment, uc.attempts)
}

func loadPaymentDetails(
	ctx context.Context,
	payment interface {
		ID() string
		Reference() string
		PayerID() string
		Purpose() enums.PaymentPurpose
		BusinessReference() value_objects.BusinessReference
		Amount() value_objects.Money
		Status() enums.PaymentStatus
		AttemptID() string
	},
	attempts ports.PaymentAttemptRepository,
) (*PaymentDetails, error) {
	ref := payment.BusinessReference()
	amount := payment.Amount()

	details := &PaymentDetails{
		ID:                    payment.ID(),
		Reference:             payment.Reference(),
		PayerID:               payment.PayerID(),
		Purpose:               payment.Purpose(),
		BusinessReferenceType: ref.Type(),
		BusinessReferenceID:   ref.ID(),
		Amount:                amount.Amount(),
		Currency:              amount.Currency(),
		Status:                payment.Status(),
	}

	if payment.AttemptID() == "" {
		return details, nil
	}

	attempt, err := attempts.GetByID(ctx, payment.AttemptID())
	if err != nil {
		return nil, fmt.Errorf("%w: %v", application.ErrPaymentAttemptNotFound, err)
	}

	details.Attempt = &PaymentAttemptDetails{
		ID:                attempt.ID(),
		PaymentID:         attempt.PaymentID(),
		Provider:          attempt.Provider(),
		ProviderReference: attempt.ProviderReference(),
		Status:            attempt.Status(),
		FailureReason:     attempt.FailureReason(),
		CheckoutURL:       attempt.CheckoutURL(),
		ExpiresAt:         attempt.ExpiresAt(),
		Metadata:          attempt.Metadata(),
	}
	return details, nil
}
