package use_cases

import (
	"context"
	"fmt"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
)

const (
	// DefaultPaymentPageSize is used when a caller omits the page size.
	DefaultPaymentPageSize = 20
	// MaxPaymentPageSize bounds the work performed by a single list request.
	MaxPaymentPageSize = 100
)

// ListPaymentsByPayerIDInput defines the payer and requested page.
type ListPaymentsByPayerIDInput struct {
	PayerID  string
	Page     int
	PageSize int
}

// ListPaymentsByPayerIDOutput contains a page and its pagination metadata.
type ListPaymentsByPayerIDOutput struct {
	Items      []*PaymentDetails
	Page       int
	PageSize   int
	Total      int64
	TotalPages int
}

// ListPaymentsByPayerIDUseCase lists a payer's payments in pages.
type ListPaymentsByPayerIDUseCase struct {
	payments ports.PaymentRepository
	attempts ports.PaymentAttemptRepository
}

// NewListPaymentsByPayerIDUseCase constructs the paginated listing use case.
func NewListPaymentsByPayerIDUseCase(
	payments ports.PaymentRepository,
	attempts ports.PaymentAttemptRepository,
) (*ListPaymentsByPayerIDUseCase, error) {
	if payments == nil || attempts == nil {
		return nil, fmt.Errorf("payment and attempt repositories are required")
	}
	return &ListPaymentsByPayerIDUseCase{
		payments: payments, attempts: attempts,
	}, nil
}

func (uc *ListPaymentsByPayerIDUseCase) Execute(
	ctx context.Context,
	input ListPaymentsByPayerIDInput,
) (*ListPaymentsByPayerIDOutput, error) {
	if err := application.ValidateUUID(input.PayerID); err != nil {
		return nil, fmt.Errorf("payer ID: %w", err)
	}

	page := input.Page
	if page == 0 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize == 0 {
		pageSize = DefaultPaymentPageSize
	}
	if page < 1 || pageSize < 1 || pageSize > MaxPaymentPageSize {
		return nil, application.ErrInvalidPagination
	}

	maxInt := int(^uint(0) >> 1)
	if page-1 > maxInt/pageSize {
		return nil, application.ErrInvalidPagination
	}
	offset := (page - 1) * pageSize

	payments, total, err := uc.payments.ListByPayerID(ctx, input.PayerID, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("list payments by payer ID: %w", err)
	}
	if total < 0 {
		return nil, fmt.Errorf("list payments by payer ID: repository returned negative total")
	}

	result := &ListPaymentsByPayerIDOutput{
		Items:      make([]*PaymentDetails, 0, len(payments)),
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: int(total / int64(pageSize)),
	}
	if total%int64(pageSize) != 0 {
		result.TotalPages++
	}

	for _, payment := range payments {
		if payment == nil {
			return nil, fmt.Errorf("list payments by payer ID: repository returned a nil payment")
		}
		if payment.PayerID() != input.PayerID {
			return nil, fmt.Errorf("list payments by payer ID: repository returned a payment outside payer scope")
		}
		details, err := loadPaymentDetails(ctx, payment, uc.attempts)
		if err != nil {
			return nil, fmt.Errorf("load listed payment details: %w", err)
		}
		result.Items = append(result.Items, details)
	}

	return result, nil
}
