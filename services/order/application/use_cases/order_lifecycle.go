package use_cases

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/domain/entities"
)

// orderLifecycle centralizes the persistence and observability shared by
// use cases that transition an existing order from one state to another.
type orderLifecycle struct {
	repository ports.OrderRepository
	logger     ports.Logger
	metrics    ports.Metrics
}

// newOrderLifecycle constructs the shared lifecycle executor used by each
// public lifecycle use case.
func newOrderLifecycle(
	repository ports.OrderRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *orderLifecycle {
	return &orderLifecycle{
		repository: repository,
		logger:     logger,
		metrics:    metrics,
	}
}

// execute loads the latest persisted order, applies a domain transition, and
// persists only when the transition succeeds.
func (uc *orderLifecycle) execute(
	ctx context.Context,
	orderID uuid.UUID,
	operation string,
	transition func(*entities.Order) error,
) error {
	// Reject an absent identifier before making a repository call.
	if orderID == uuid.Nil {
		return fmt.Errorf("%s: order ID is required", operation)
	}

	order, err := uc.repository.FindByID(ctx, orderID)
	if err != nil {
		uc.log(ctx, operation, "load_order")
		return fmt.Errorf("%s: load order: %w", operation, err)
	}
	if order == nil {
		uc.log(ctx, operation, "load_order")
		return fmt.Errorf("%s: repository returned no order", operation)
	}
	if order.ID != orderID {
		uc.log(ctx, operation, "load_order")
		return fmt.Errorf("%s: repository returned a different order", operation)
	}

	// Domain entities remain the authority for authorization and valid states.
	if err := transition(order); err != nil {
		uc.log(ctx, operation, "domain_transition")
		return fmt.Errorf("%s: transition order: %w", operation, err)
	}

	// Never persist an order whose domain transition failed.
	if err := uc.repository.Update(ctx, order); err != nil {
		uc.log(ctx, operation, "persistence")
		return fmt.Errorf("%s: persist order: %w", operation, err)
	}

	// Observability failures must not turn a successfully persisted transition
	// into a reported business failure.
	uc.log(ctx, operation, "")
	_ = uc.metrics.Increment(ctx, ports.Metric{
		Name:   "order_lifecycle_transitions",
		Value:  1,
		Labels: map[string]string{"operation": operation},
	})
	return nil
}

// log records a lifecycle event without allowing logging failures to affect
// the order operation.
func (uc *orderLifecycle) log(
	ctx context.Context,
	operation string,
	failureCategory string,
) {
	_ = uc.logger.Log(ctx, ports.LogEvent{
		Event:           "order.lifecycle",
		Operation:       operation,
		FailureCategory: failureCategory,
	})
}

// AddShippingInput contains the owning store and proposed shipping amount.
type AddShippingInput struct {
	OrderID        uuid.UUID
	StoreID        uuid.UUID
	ShippingAmount int64
}

// AddShipping proposes or revises shipping for an eligible order.
type AddShipping struct {
	lifecycle *orderLifecycle
}

// NewAddShipping constructs the AddShipping use case.
func NewAddShipping(
	repository ports.OrderRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *AddShipping {
	return &AddShipping{lifecycle: newOrderLifecycle(repository, logger, metrics)}
}

// Execute applies the domain's store-ownership and state-transition rules.
func (uc *AddShipping) Execute(ctx context.Context, input AddShippingInput) error {
	return uc.lifecycle.execute(ctx, input.OrderID, "add_shipping", func(order *entities.Order) error {
		return order.AddShipping(input.StoreID, input.ShippingAmount)
	})
}

// RejectShippingInput identifies the order and customer rejecting its proposal.
type RejectShippingInput struct {
	OrderID    uuid.UUID
	CustomerID uuid.UUID
}

// RejectShipping records a customer's rejection of the proposed shipping.
type RejectShipping struct {
	lifecycle *orderLifecycle
}

// NewRejectShipping constructs the RejectShipping use case.
func NewRejectShipping(
	repository ports.OrderRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *RejectShipping {
	return &RejectShipping{lifecycle: newOrderLifecycle(repository, logger, metrics)}
}

// Execute delegates customer authorization and state validation to the domain.
func (uc *RejectShipping) Execute(ctx context.Context, input RejectShippingInput) error {
	return uc.lifecycle.execute(ctx, input.OrderID, "reject_shipping", func(order *entities.Order) error {
		return order.RejectShipping(input.CustomerID)
	})
}

// MarkPaidInput identifies the order whose payment has been confirmed.
type MarkPaidInput struct {
	OrderID uuid.UUID
}

// MarkPaid records trusted payment confirmation from the Payment service.
// The calling adapter must authenticate the integration and establish that
// payment has actually succeeded before invoking this use case.
type MarkPaid struct {
	lifecycle *orderLifecycle
}

// NewMarkPaid constructs the MarkPaid use case.
func NewMarkPaid(
	repository ports.OrderRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *MarkPaid {
	return &MarkPaid{lifecycle: newOrderLifecycle(repository, logger, metrics)}
}

// Execute marks an eligible order as paid after trusted payment confirmation.
func (uc *MarkPaid) Execute(ctx context.Context, input MarkPaidInput) error {
	return uc.lifecycle.execute(ctx, input.OrderID, "mark_paid", func(order *entities.Order) error {
		return order.MarkPaid()
	})
}

// CancelOrderInput identifies the order and actor requesting cancellation.
type CancelOrderInput struct {
	OrderID uuid.UUID
	ActorID uuid.UUID
}

// CancelOrder cancels an order when its current state permits cancellation.
type CancelOrder struct {
	lifecycle *orderLifecycle
}

// NewCancelOrder constructs the CancelOrder use case.
func NewCancelOrder(
	repository ports.OrderRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *CancelOrder {
	return &CancelOrder{lifecycle: newOrderLifecycle(repository, logger, metrics)}
}

// Execute delegates actor authorization and cancellation rules to the domain.
func (uc *CancelOrder) Execute(ctx context.Context, input CancelOrderInput) error {
	return uc.lifecycle.execute(ctx, input.OrderID, "cancel_order", func(order *entities.Order) error {
		return order.Cancel(input.ActorID)
	})
}

// MarkSentInput identifies the order and store dispatching it.
type MarkSentInput struct {
	OrderID uuid.UUID
	StoreID uuid.UUID
}

// MarkSent records dispatch by the store that owns the order.
type MarkSent struct {
	lifecycle *orderLifecycle
}

// NewMarkSent constructs the MarkSent use case.
func NewMarkSent(
	repository ports.OrderRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *MarkSent {
	return &MarkSent{lifecycle: newOrderLifecycle(repository, logger, metrics)}
}

// Execute delegates store authorization and paid-state validation to the domain.
func (uc *MarkSent) Execute(ctx context.Context, input MarkSentInput) error {
	return uc.lifecycle.execute(ctx, input.OrderID, "mark_sent", func(order *entities.Order) error {
		return order.MarkSent(input.StoreID)
	})
}

// ConfirmReceiptInput identifies the order and customer confirming delivery.
type ConfirmReceiptInput struct {
	OrderID    uuid.UUID
	CustomerID uuid.UUID
}

// ConfirmReceipt records the customer's confirmation that the order arrived.
type ConfirmReceipt struct {
	lifecycle *orderLifecycle
}

// NewConfirmReceipt constructs the ConfirmReceipt use case.
func NewConfirmReceipt(
	repository ports.OrderRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *ConfirmReceipt {
	return &ConfirmReceipt{lifecycle: newOrderLifecycle(repository, logger, metrics)}
}

// Execute delegates customer authorization and sent-state validation to the domain.
func (uc *ConfirmReceipt) Execute(ctx context.Context, input ConfirmReceiptInput) error {
	return uc.lifecycle.execute(ctx, input.OrderID, "confirm_receipt", func(order *entities.Order) error {
		return order.ConfirmReceipt(input.CustomerID)
	})
}
