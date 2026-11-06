package use_cases

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
)

// SubscribeInput contains the information required to subscribe a store.
type SubscribeInput struct {
	StoreID       uuid.UUID
	PayerID       uuid.UUID
	PlanID        uuid.UUID
	CustomerEmail string
}

// SubscribeOutput contains the pending subscription and its checkout details.
type SubscribeOutput struct {
	Subscription entities.Subscription
	Payment      ports.InitiatePaymentResult
}

// Subscribe creates or resumes a store subscription and initiates its payment.
//
// A pending subscription for the same plan is reused. A pending subscription
// for another plan must be cancelled before a different plan can be selected.
// An active subscription prevents a new subscription from being created.
//
// Subscription activation is deliberately outside this use case. It must
// happen only after Billing receives a trusted, verified payment event.
type Subscribe struct {
	plans         ports.PlanRepository
	subscriptions ports.SubscriptionRepository
	payment       ports.Payment
	unitOfWork    ports.UnitOfWork
	outbox        ports.SubscriptionOutboxRepository
	telemetry     operationTelemetry
}

// NewSubscribe constructs the subscription use case.
// Observability dependencies are optional.
func NewSubscribe(
	plans ports.PlanRepository,
	subscriptions ports.SubscriptionRepository,
	payment ports.Payment,
	unitOfWork ports.UnitOfWork,
	outbox ports.SubscriptionOutboxRepository,
	observability ...ports.Observability,
) *Subscribe {
	var dependencies ports.Observability
	if len(observability) > 0 {
		dependencies = observability[0]
	}

	return &Subscribe{
		plans:         plans,
		subscriptions: subscriptions,
		payment:       payment,
		unitOfWork:    unitOfWork,
		outbox:        outbox,
		telemetry: newOperationTelemetry(
			dependencies.Logger,
			dependencies.Metrics,
			"subscribe",
			"billing_subscribe",
		),
	}
}

// Execute creates or resumes a pending subscription and requests checkout.
func (useCase *Subscribe) Execute(
	ctx context.Context,
	input SubscribeInput,
) (SubscribeOutput, error) {
	if useCase == nil {
		return SubscribeOutput{}, application.ErrSubscriptionRepositoryUnavailable
	}

	startedAt := useCase.telemetry.start(ctx)

	fail := func(category string, err error) (SubscribeOutput, error) {
		useCase.telemetry.failure(ctx, startedAt, category)
		return SubscribeOutput{}, err
	}

	if useCase.subscriptions == nil {
		return fail("repository_unavailable", application.ErrSubscriptionRepositoryUnavailable)
	}
	if useCase.plans == nil {
		return fail("repository_unavailable", application.ErrPlanRepositoryUnavailable)
	}
	if useCase.payment == nil {
		return fail("repository_unavailable", application.ErrPaymentUnavailable)
	}
	if useCase.unitOfWork == nil {
		return fail("unit_of_work_unavailable", fmt.Errorf("subscription unit of work is unavailable"))
	}
	if useCase.outbox == nil {
		return fail("outbox_unavailable", fmt.Errorf("subscription outbox repository is unavailable"))
	}

	if input.StoreID == uuid.Nil ||
		input.PayerID == uuid.Nil ||
		input.PlanID == uuid.Nil {
		return fail("validation", domain.ErrInvalidSubscription)
	}

	email := strings.TrimSpace(input.CustomerEmail)
	parsedEmail, emailErr := mail.ParseAddress(email)
	if emailErr != nil || parsedEmail.Address != email {
		return fail("validation", domain.ErrInvalidSubscription)
	}

	// Resume an existing pending subscription before looking up the plan.
	// Its original commercial terms must remain unchanged even if the plan
	// has since been edited or made unavailable.
	existing, err := useCase.subscriptions.GetCurrentByStore(ctx, input.StoreID)
	if err != nil && !errors.Is(err, application.ErrSubscriptionNotFound) {
		return fail("retrieval", err)
	}

	if existing != nil {
		output, resumeErr := useCase.resumeOrReject(
			ctx,
			*existing,
			input,
			email,
		)
		if resumeErr != nil {
			return fail("retrieval", resumeErr)
		}

		useCase.telemetry.success(ctx, startedAt)
		return output, nil
	}

	plan, err := useCase.plans.GetByID(ctx, input.PlanID)
	if err != nil {
		return fail("retrieval", err)
	}
	if !plan.Available {
		return fail("validation", application.ErrPlanUnavailable)
	}
	if err := plan.Validate(); err != nil {
		return fail("validation", err)
	}

	now := time.Now().UTC()

	// Snapshot the effective price, including any promotion, into the
	// subscription. Later plan changes must not change this payment amount.
	price, err := plan.PriceAt(now)
	if err != nil {
		return fail("validation", err)
	}
	plan.Price = price

	subscription, err := entities.NewSubscription(
		uuid.New(),
		input.StoreID,
		plan,
		now,
	)
	if err != nil {
		return fail("validation", err)
	}

	if err := useCase.subscriptions.Create(ctx, *subscription); err != nil {
		if errors.Is(err, application.ErrSubscriptionConflict) {
			// Another request may have created a subscription after our
			// initial lookup. Reload it and apply the same business rules.
			existing, lookupErr := useCase.subscriptions.GetCurrentByStore(
				ctx,
				input.StoreID,
			)
			if lookupErr != nil {
				return fail("retrieval", lookupErr)
			}
			if existing == nil {
				return fail("persistence", application.ErrSubscriptionConflict)
			}

			output, resumeErr := useCase.resumeOrReject(
				ctx,
				*existing,
				input,
				email,
			)
			if resumeErr != nil {
				return fail("retrieval", resumeErr)
			}

			useCase.telemetry.success(ctx, startedAt)
			return output, nil
		}

		return fail("persistence", err)
	}

	output, err := useCase.initiatePayment(ctx, subscription, input.PayerID, email)
	if err != nil {
		return fail("persistence", err)
	}

	useCase.telemetry.success(ctx, startedAt)
	return output, nil
}

// resumeOrReject enforces the active/pending subscription rules.
func (useCase *Subscribe) resumeOrReject(
	ctx context.Context,
	subscription entities.Subscription,
	input SubscribeInput,
	email string,
) (SubscribeOutput, error) {
	switch subscription.Status {
	case entities.SubscriptionStatusActive:
		return SubscribeOutput{}, application.ErrActiveSubscriptionExists

	case entities.SubscriptionStatusPending:
		if subscription.PlanID != input.PlanID {
			return SubscribeOutput{}, application.ErrSubscriptionAlreadyPending
		}

		return useCase.initiatePayment(
			ctx,
			&subscription,
			input.PayerID,
			email,
		)

	default:
		return SubscribeOutput{}, application.ErrSubscriptionConflict
	}
}

// initiatePayment calls Payment using the subscription ID as the stable
// business reference. Repeated requests for the same pending subscription
// therefore use the same reference.
func (useCase *Subscribe) initiatePayment(
	ctx context.Context,
	subscription *entities.Subscription,
	payerID uuid.UUID,
	email string,
) (SubscribeOutput, error) {
	result, err := useCase.payment.InitiatePayment(
		ctx,
		ports.InitiatePaymentRequest{
			PayerID:               payerID.String(),
			Purpose:               ports.PaymentPurposeSubscription,
			BusinessReferenceType: ports.BusinessReferenceTypeSubscription,
			BusinessReferenceID:   subscription.ID.String(),
			AmountMinor:           subscription.Price.AmountMinor,
			Currency:              subscription.Price.Currency,
			CustomerEmail:         email,
		},
	)
	if err != nil {
		return SubscribeOutput{}, useCase.markPaymentInitiationFailed(
			ctx,
			subscription,
			err,
		)
	}

	if strings.TrimSpace(result.PaymentID) == "" ||
		strings.TrimSpace(result.CheckoutURL) == "" {
		return SubscribeOutput{}, useCase.markPaymentInitiationFailed(
			ctx,
			subscription,
			application.ErrInvalidPaymentResponse,
		)
	}

	return SubscribeOutput{
		Subscription: *subscription,
		Payment:      result,
	}, nil
}

// markPaymentInitiationFailed releases the pending subscription after a
// failed payment initiation so the store can make another subscription request.
func (useCase *Subscribe) markPaymentInitiationFailed(
	ctx context.Context,
	subscription *entities.Subscription,
	cause error,
) error {
	err := useCase.unitOfWork.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := subscription.MarkPaymentFailed(time.Now().UTC()); err != nil {
			return fmt.Errorf("mark subscription payment failed: %w", err)
		}
		if err := useCase.subscriptions.Update(txCtx, *subscription); err != nil {
			return fmt.Errorf("persist failed subscription: %w", err)
		}
		if err := recordSubscriptionUpdate(txCtx, useCase.outbox, useCase.plans, *subscription); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return errors.Join(
			application.ErrPaymentInitiationFailed,
			cause,
			fmt.Errorf("persist failed subscription and outbox event: %w", err),
		)
	}

	return errors.Join(application.ErrPaymentInitiationFailed, cause)
}
