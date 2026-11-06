package use_cases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/value_objects"
)

func newSubscribePlan(t *testing.T) entities.Plan {
	t.Helper()

	price, err := value_objects.NewMoney(1200000, "NGN")
	if err != nil {
		t.Fatal(err)
	}

	return entities.Plan{
		ID:           uuid.New(),
		Type:         "premium",
		Name:         "Premium",
		Price:        price,
		Duration:     30 * 24 * time.Hour,
		Capabilities: entities.PlanCapabilities{MaxProducts: 5, AIEnabled: true},
		Available:    true,
	}
}

func newSubscribeUseCase(
	plans ports.PlanRepository,
	subscriptions ports.SubscriptionRepository,
	payment ports.Payment,
) *use_cases.Subscribe {
	return use_cases.NewSubscribe(
		plans,
		subscriptions,
		payment,
		&fakeUnitOfWork{},
		&fakeSubscriptionOutboxRepository{},
	)
}

func newSubscribePayment() *fakeSubscriptionPayment {
	return &fakeSubscriptionPayment{
		result: ports.InitiatePaymentResult{
			PaymentID:   uuid.NewString(),
			AttemptID:   uuid.NewString(),
			CheckoutURL: "https://checkout.example.test/payment",
			Provider:    "paystack",
			AmountMinor: 1200000,
			Currency:    "NGN",
		},
	}
}

func TestSubscribeCreatesPendingSubscriptionAndInitiatesPayment(t *testing.T) {
	plan := newSubscribePlan(t)
	planRepository := &fakePlanRepository{plan: plan}
	subscriptionRepository := &fakeSubscriptionRepository{}
	payment := newSubscribePayment()

	useCase := newSubscribeUseCase(
		planRepository,
		subscriptionRepository,
		payment,
	)

	input := use_cases.SubscribeInput{
		StoreID:       uuid.New(),
		PayerID:       uuid.New(),
		PlanID:        plan.ID,
		CustomerEmail: "owner@example.com",
	}

	output, err := useCase.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if output.Subscription.Status != entities.SubscriptionStatusPending {
		t.Fatalf("expected pending subscription, got %q", output.Subscription.Status)
	}
	if output.Subscription.StoreID != input.StoreID {
		t.Fatal("subscription store ID does not match input")
	}
	if output.Subscription.PlanID != plan.ID {
		t.Fatal("subscription plan ID does not match input")
	}
	if output.Subscription.Price != plan.Price {
		t.Fatal("subscription did not snapshot the plan price")
	}
	if !output.Subscription.StartedAt.IsZero() || !output.Subscription.ExpiresAt.IsZero() {
		t.Fatal("new pending subscription must not have active dates")
	}
	if subscriptionRepository.created == nil {
		t.Fatal("expected subscription to be persisted")
	}
	if payment.calls != 1 {
		t.Fatalf("expected one payment initiation, got %d", payment.calls)
	}
	if payment.request.BusinessReferenceID != output.Subscription.ID.String() {
		t.Fatal("payment business reference must be the subscription ID")
	}
	if payment.request.BusinessReferenceType != ports.BusinessReferenceTypeSubscription {
		t.Fatalf("unexpected business reference type: %q", payment.request.BusinessReferenceType)
	}
	if payment.request.Purpose != ports.PaymentPurposeSubscription {
		t.Fatalf("unexpected payment purpose: %q", payment.request.Purpose)
	}
	if payment.request.AmountMinor != plan.Price.AmountMinor ||
		payment.request.Currency != plan.Price.Currency {
		t.Fatal("payment request does not match the subscription's price snapshot")
	}
	if output.Payment.CheckoutURL == "" {
		t.Fatal("expected checkout URL in output")
	}
}

func TestSubscribeRejectsUnavailablePlan(t *testing.T) {
	plan := newSubscribePlan(t)
	plan.Available = false

	repository := &fakeSubscriptionRepository{}
	payment := newSubscribePayment()
	useCase := newSubscribeUseCase(
		&fakePlanRepository{plan: plan},
		repository,
		payment,
	)

	_, err := useCase.Execute(context.Background(), use_cases.SubscribeInput{
		StoreID:       uuid.New(),
		PayerID:       uuid.New(),
		PlanID:        plan.ID,
		CustomerEmail: "owner@example.com",
	})

	if !errors.Is(err, application.ErrPlanUnavailable) {
		t.Fatalf("expected ErrPlanUnavailable, got %v", err)
	}
	if repository.created != nil || payment.calls != 0 {
		t.Fatal("unavailable plan must not create a subscription or initiate payment")
	}
}

func TestSubscribeRejectsExistingActiveSubscription(t *testing.T) {
	plan := newSubscribePlan(t)
	existing := entities.Subscription{
		ID:      uuid.New(),
		StoreID: uuid.New(),
		PlanID:  plan.ID,
		Status:  entities.SubscriptionStatusActive,
	}

	repository := &fakeSubscriptionRepository{current: &existing}
	payment := newSubscribePayment()
	useCase := newSubscribeUseCase(
		&fakePlanRepository{plan: plan},
		repository,
		payment,
	)

	_, err := useCase.Execute(context.Background(), use_cases.SubscribeInput{
		StoreID:       existing.StoreID,
		PayerID:       uuid.New(),
		PlanID:        plan.ID,
		CustomerEmail: "owner@example.com",
	})

	if !errors.Is(err, application.ErrActiveSubscriptionExists) {
		t.Fatalf("expected ErrActiveSubscriptionExists, got %v", err)
	}
	if payment.calls != 0 {
		t.Fatal("must not initiate payment for a store with an active subscription")
	}
}

func TestSubscribeResumesPendingSubscriptionForSamePlan(t *testing.T) {
	plan := newSubscribePlan(t)
	originalPrice, err := value_objects.NewMoney(950000, "NGN")
	if err != nil {
		t.Fatal(err)
	}

	existing := entities.Subscription{
		ID:       uuid.New(),
		StoreID:  uuid.New(),
		PlanID:   plan.ID,
		Price:    originalPrice,
		Duration: 30 * 24 * time.Hour,
		Status:   entities.SubscriptionStatusPending,
	}

	repository := &fakeSubscriptionRepository{current: &existing}
	payment := newSubscribePayment()
	payment.result.AmountMinor = originalPrice.AmountMinor

	useCase := newSubscribeUseCase(
		&fakePlanRepository{plan: plan},
		repository,
		payment,
	)

	output, err := useCase.Execute(context.Background(), use_cases.SubscribeInput{
		StoreID:       existing.StoreID,
		PayerID:       uuid.New(),
		PlanID:        plan.ID,
		CustomerEmail: "owner@example.com",
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if output.Subscription.ID != existing.ID {
		t.Fatal("expected the existing subscription ID to be reused")
	}
	if output.Subscription.Price != originalPrice {
		t.Fatal("retry must preserve the original subscription price")
	}
	if repository.created != nil {
		t.Fatal("retry must not create another subscription")
	}
	if payment.calls != 1 {
		t.Fatalf("expected one payment retry, got %d", payment.calls)
	}
	if payment.request.BusinessReferenceID != existing.ID.String() {
		t.Fatal("retry must use the existing subscription as business reference")
	}
	if payment.request.AmountMinor != originalPrice.AmountMinor {
		t.Fatal("retry must use the original subscription amount")
	}
}

func TestSubscribeRejectsDifferentPlanWhileSubscriptionIsPending(t *testing.T) {
	plan := newSubscribePlan(t)
	existing := entities.Subscription{
		ID:      uuid.New(),
		StoreID: uuid.New(),
		PlanID:  uuid.New(),
		Status:  entities.SubscriptionStatusPending,
	}

	repository := &fakeSubscriptionRepository{current: &existing}
	payment := newSubscribePayment()
	useCase := newSubscribeUseCase(
		&fakePlanRepository{plan: plan},
		repository,
		payment,
	)

	_, err := useCase.Execute(context.Background(), use_cases.SubscribeInput{
		StoreID:       existing.StoreID,
		PayerID:       uuid.New(),
		PlanID:        plan.ID,
		CustomerEmail: "owner@example.com",
	})

	if !errors.Is(err, application.ErrSubscriptionAlreadyPending) {
		t.Fatalf("expected ErrSubscriptionAlreadyPending, got %v", err)
	}
	if payment.calls != 0 {
		t.Fatal("must not initiate payment for a different plan while one is pending")
	}
}

func TestSubscribeMarksSubscriptionFailedWhenPaymentInitiationFails(t *testing.T) {
	plan := newSubscribePlan(t)
	paymentErr := errors.New("payment service failed")
	repository := &fakeSubscriptionRepository{}
	payment := &fakeSubscriptionPayment{err: paymentErr}

	useCase := newSubscribeUseCase(
		&fakePlanRepository{plan: plan},
		repository,
		payment,
	)

	_, err := useCase.Execute(context.Background(), use_cases.SubscribeInput{
		StoreID:       uuid.New(),
		PayerID:       uuid.New(),
		PlanID:        plan.ID,
		CustomerEmail: "owner@example.com",
	})

	if !errors.Is(err, application.ErrPaymentInitiationFailed) {
		t.Fatalf("expected ErrPaymentInitiationFailed, got %v", err)
	}
	if !errors.Is(err, paymentErr) {
		t.Fatalf("expected original payment error to be preserved, got %v", err)
	}
	if repository.updated == nil {
		t.Fatal("expected failed subscription to be persisted")
	}
	if repository.updated.Status != entities.SubscriptionStatusFailed {
		t.Fatalf("expected failed subscription, got %q", repository.updated.Status)
	}
}

func TestSubscribeMarksSubscriptionFailedWhenPaymentResponseIsInvalid(t *testing.T) {
	plan := newSubscribePlan(t)
	repository := &fakeSubscriptionRepository{}
	payment := &fakeSubscriptionPayment{
		result: ports.InitiatePaymentResult{
			PaymentID: uuid.NewString(),
		},
	}

	useCase := newSubscribeUseCase(
		&fakePlanRepository{plan: plan},
		repository,
		payment,
	)

	_, err := useCase.Execute(context.Background(), use_cases.SubscribeInput{
		StoreID:       uuid.New(),
		PayerID:       uuid.New(),
		PlanID:        plan.ID,
		CustomerEmail: "owner@example.com",
	})

	if !errors.Is(err, application.ErrPaymentInitiationFailed) {
		t.Fatalf("expected ErrPaymentInitiationFailed, got %v", err)
	}
	if !errors.Is(err, application.ErrInvalidPaymentResponse) {
		t.Fatalf("expected ErrInvalidPaymentResponse, got %v", err)
	}
	if repository.updated == nil ||
		repository.updated.Status != entities.SubscriptionStatusFailed {
		t.Fatal("expected invalid payment response to mark subscription failed")
	}
}

func TestSubscribeRejectsInvalidInput(t *testing.T) {
	plan := newSubscribePlan(t)
	repository := &fakeSubscriptionRepository{}
	payment := newSubscribePayment()

	useCase := newSubscribeUseCase(
		&fakePlanRepository{plan: plan},
		repository,
		payment,
	)

	_, err := useCase.Execute(context.Background(), use_cases.SubscribeInput{
		StoreID:       uuid.Nil,
		PayerID:       uuid.New(),
		PlanID:        plan.ID,
		CustomerEmail: "owner@example.com",
	})

	if err == nil {
		t.Fatal("expected invalid input error")
	}
	if repository.created != nil || payment.calls != 0 {
		t.Fatal("invalid input must not create a subscription or initiate payment")
	}
}

func TestSubscribeRejectsMissingPaymentDependency(t *testing.T) {
	plan := newSubscribePlan(t)
	useCase := newSubscribeUseCase(
		&fakePlanRepository{plan: plan},
		&fakeSubscriptionRepository{},
		nil,
	)

	_, err := useCase.Execute(context.Background(), use_cases.SubscribeInput{
		StoreID:       uuid.New(),
		PayerID:       uuid.New(),
		PlanID:        plan.ID,
		CustomerEmail: "owner@example.com",
	})

	if !errors.Is(err, application.ErrPaymentUnavailable) {
		t.Fatalf("expected ErrPaymentUnavailable, got %v", err)
	}
}

func TestSubscribeRecordsFailedPaymentInOutboxWithinTransaction(t *testing.T) {
	plan := newSubscribePlan(t)
	repository := &fakeSubscriptionRepository{}
	payment := &fakeSubscriptionPayment{
		err: errors.New("payment service unavailable"),
	}
	uow := &fakeUnitOfWork{}
	outbox := &fakeSubscriptionOutboxRepository{}

	useCase := use_cases.NewSubscribe(
		&fakePlanRepository{plan: plan},
		repository,
		payment,
		uow,
		outbox,
	)

	_, err := useCase.Execute(context.Background(), use_cases.SubscribeInput{
		StoreID:       uuid.New(),
		PayerID:       uuid.New(),
		PlanID:        plan.ID,
		CustomerEmail: "owner@example.com",
	})
	if !errors.Is(err, application.ErrPaymentInitiationFailed) {
		t.Fatalf("error = %v, want ErrPaymentInitiationFailed", err)
	}
	if repository.updated == nil ||
		repository.updated.Status != entities.SubscriptionStatusFailed {
		t.Fatal("failed subscription was not persisted")
	}
	if uow.calls != 1 || outbox.calls != 1 {
		t.Fatalf("transaction calls = %d, outbox calls = %d; want 1 each", uow.calls, outbox.calls)
	}
	if outbox.subscriptionID != repository.updated.ID.String() {
		t.Fatal("outbox event references the wrong subscription")
	}
	if outbox.eventType != "billing.subscription.updated" {
		t.Fatalf("event type = %q", outbox.eventType)
	}
}

func TestSubscribeReturnsErrorWhenFailedPaymentOutboxWriteFails(t *testing.T) {
	plan := newSubscribePlan(t)
	repository := &fakeSubscriptionRepository{}
	payment := &fakeSubscriptionPayment{
		err: errors.New("payment service unavailable"),
	}
	uow := &fakeUnitOfWork{}
	outbox := &fakeSubscriptionOutboxRepository{
		err: errors.New("outbox unavailable"),
	}

	useCase := use_cases.NewSubscribe(
		&fakePlanRepository{plan: plan},
		repository,
		payment,
		uow,
		outbox,
	)

	_, err := useCase.Execute(context.Background(), use_cases.SubscribeInput{
		StoreID:       uuid.New(),
		PayerID:       uuid.New(),
		PlanID:        plan.ID,
		CustomerEmail: "owner@example.com",
	})
	if !errors.Is(err, application.ErrPaymentInitiationFailed) {
		t.Fatalf("error = %v, want ErrPaymentInitiationFailed", err)
	}
	if outbox.calls != 1 {
		t.Fatalf("outbox calls = %d, want 1", outbox.calls)
	}
}

func TestSubscribeDoesNotUseTransactionForSuccessfulPaymentInitiation(t *testing.T) {
	plan := newSubscribePlan(t)
	repository := &fakeSubscriptionRepository{}
	payment := newSubscribePayment()
	uow := &fakeUnitOfWork{}
	outbox := &fakeSubscriptionOutboxRepository{}

	useCase := use_cases.NewSubscribe(
		&fakePlanRepository{plan: plan},
		repository,
		payment,
		uow,
		outbox,
	)

	_, err := useCase.Execute(context.Background(), use_cases.SubscribeInput{
		StoreID:       uuid.New(),
		PayerID:       uuid.New(),
		PlanID:        plan.ID,
		CustomerEmail: "owner@example.com",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if uow.calls != 0 || outbox.calls != 0 {
		t.Fatal("successful payment initiation must not open a transaction or write a Store event")
	}
}
