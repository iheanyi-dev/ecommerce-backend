package use_cases

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/domain/entities"
)

// InitiatePaymentInput identifies the order and authenticated customer.
type InitiatePaymentInput struct {
	OrderID    uuid.UUID
	CustomerID uuid.UUID
}

// InitiatePaymentOutput contains the Payment service response needed by checkout.
type InitiatePaymentOutput struct {
	PaymentID   string
	AttemptID   string
	CheckoutURL string
}

// InitiatePayment starts payment for an eligible order owned by the customer.
type InitiatePayment struct {
	orders   ports.OrderRepository
	identity ports.IdentityClient
	payment  ports.PaymentClient
	logger   ports.Logger
	metrics  ports.Metrics
}

// NewInitiatePayment constructs the InitiatePayment use case.
func NewInitiatePayment(
	orders ports.OrderRepository,
	identity ports.IdentityClient,
	payment ports.PaymentClient,
	logger ports.Logger,
	metrics ports.Metrics,
) *InitiatePayment {
	return &InitiatePayment{
		orders:   orders,
		identity: identity,
		payment:  payment,
		logger:   logger,
		metrics:  metrics,
	}
}

// Execute retrieves the customer's email from Identity and requests payment
// using the order's product subtotal plus shipping. It deliberately does not
// change the order state; trusted Payment events do that.
func (uc *InitiatePayment) Execute(
	ctx context.Context,
	input InitiatePaymentInput,
) (*InitiatePaymentOutput, error) {
	const operation = "initiate_payment"

	if input.OrderID == uuid.Nil {
		return nil, fmt.Errorf("%s: order ID is required", operation)
	}
	if input.CustomerID == uuid.Nil {
		return nil, fmt.Errorf("%s: customer ID is required", operation)
	}

	order, err := uc.orders.FindByID(ctx, input.OrderID)
	if err != nil {
		uc.log(ctx, operation, "load_order")
		return nil, fmt.Errorf("%s: load order: %w", operation, err)
	}
	if order == nil || order.ID != input.OrderID {
		uc.log(ctx, operation, "load_order")
		return nil, fmt.Errorf("%s: order not found", operation)
	}
	if order.CustomerID != input.CustomerID {
		uc.log(ctx, operation, "order_ownership")
		return nil, fmt.Errorf("%s: order does not belong to the authenticated customer", operation)
	}
	if order.State != entities.OrderStateShippingAdded &&
		order.State != entities.OrderStatePaymentFailed {
		uc.log(ctx, operation, "invalid_order_state")
		return nil, fmt.Errorf("%s: order state %q does not allow payment initiation",
			operation, order.State)
	}

	total, err := order.TotalPrice()
	if err != nil {
		uc.log(ctx, operation, "calculate_total")
		return nil, fmt.Errorf("%s: calculate order total: %w", operation, err)
	}
	if total <= 0 {
		uc.log(ctx, operation, "invalid_amount")
		return nil, fmt.Errorf("%s: order total must be positive", operation)
	}

	email, err := uc.identity.GetUserEmail(ctx, input.CustomerID)
	if err != nil {
		uc.log(ctx, operation, "identity_service")
		return nil, fmt.Errorf("%s: retrieve customer email from Identity service: %w", operation, err)
	}
	email = strings.TrimSpace(email)
	if email == "" {
		uc.log(ctx, operation, "invalid_customer_email")
		return nil, fmt.Errorf("%s: Identity service returned an empty customer email", operation)
	}

	response, err := uc.payment.InitiatePayment(ctx, ports.InitiatePaymentRequest{
		PayerID:               order.CustomerID,
		Purpose:               "order",
		BusinessReferenceType: "order",
		BusinessReferenceID:   order.ID,
		AmountMinor:           total,
		Currency:              order.Currency,
		CustomerEmail:         email,
	})
	if err != nil {
		uc.log(ctx, operation, "payment_service")
		return nil, fmt.Errorf("%s: initiate payment through Payment service: %w", operation, err)
	}
	if response == nil || strings.TrimSpace(response.CheckoutURL) == "" {
		uc.log(ctx, operation, "invalid_payment_response")
		return nil, fmt.Errorf("%s: Payment service returned no checkout URL", operation)
	}

	uc.log(ctx, operation, "")
	_ = uc.metrics.Increment(ctx, ports.Metric{
		Name:   "order_payments_initiated",
		Value:  1,
		Labels: map[string]string{"operation": operation},
	})

	return &InitiatePaymentOutput{
		PaymentID:   response.PaymentID,
		AttemptID:   response.AttemptID,
		CheckoutURL: response.CheckoutURL,
	}, nil
}

// log records outcomes without allowing observability failures to affect payment.
func (uc *InitiatePayment) log(ctx context.Context, operation, failureCategory string) {
	_ = uc.logger.Log(ctx, ports.LogEvent{
		Event:           "order.payment",
		Operation:       operation,
		FailureCategory: failureCategory,
	})
}
