package use_cases_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/value_objects"
)

const (
	testPayerID = "11111111-1111-4111-8111-111111111111"
	testOrderID = "22222222-2222-4222-8222-222222222222"
)

type paymentRepo struct {
	items   map[string]*entities.Payment
	saveErr error
}

func newPaymentRepo() *paymentRepo {
	return &paymentRepo{items: make(map[string]*entities.Payment)}
}

func businessKey(ref value_objects.BusinessReference) string {
	return ref.Type() + ":" + ref.ID()
}

func (r *paymentRepo) GetByID(_ context.Context, id string) (*entities.Payment, error) {
	for _, payment := range r.items {
		if payment.ID() == id {
			return payment, nil
		}
	}
	return nil, application.ErrPaymentNotFound
}

func (r *paymentRepo) GetByBusinessReference(_ context.Context, ref value_objects.BusinessReference) (*entities.Payment, error) {
	payment, ok := r.items[businessKey(ref)]
	if !ok {
		return nil, application.ErrPaymentNotFound
	}
	return payment, nil
}

func (r *paymentRepo) ListByPayerID(_ context.Context, payerID string, limit, offset int) ([]*entities.Payment, int64, error) {
	var matching []*entities.Payment
	for _, payment := range r.items {
		if payment.PayerID() == payerID {
			matching = append(matching, payment)
		}
	}
	// Deterministic order for pagination tests.
	for i := 0; i < len(matching); i++ {
		for j := i + 1; j < len(matching); j++ {
			if matching[j].CreatedAt().Before(matching[i].CreatedAt()) ||
				(matching[j].CreatedAt().Equal(matching[i].CreatedAt()) && matching[j].ID() < matching[i].ID()) {
				matching[i], matching[j] = matching[j], matching[i]
			}
		}
	}
	total := int64(len(matching))
	if offset >= len(matching) {
		return []*entities.Payment{}, total, nil
	}
	end := offset + limit
	if end > len(matching) {
		end = len(matching)
	}
	return matching[offset:end], total, nil
}

func (r *paymentRepo) Save(_ context.Context, payment *entities.Payment) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.items[businessKey(payment.BusinessReference())] = payment
	return nil
}

type attemptRepo struct {
	items   map[string]*entities.PaymentAttempt
	saveErr error
}

func newAttemptRepo() *attemptRepo {
	return &attemptRepo{items: make(map[string]*entities.PaymentAttempt)}
}

func (r *attemptRepo) GetByID(_ context.Context, id string) (*entities.PaymentAttempt, error) {
	attempt, ok := r.items[id]
	if !ok {
		return nil, application.ErrPaymentAttemptNotFound
	}
	return attempt, nil
}

func (r *attemptRepo) Save(_ context.Context, attempt *entities.PaymentAttempt) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.items[attempt.ID()] = attempt
	return nil
}

type fakeProvider struct {
	checkoutResult  ports.CheckoutResult
	checkoutErr     error
	reconcileResult ports.ReconciliationResult
	reconcileErr    error
	checkoutCalls   int
	reconcileCalls  int
}

func (p *fakeProvider) InitiateCheckout(_ context.Context, _ ports.CheckoutRequest) (ports.CheckoutResult, error) {
	p.checkoutCalls++
	return p.checkoutResult, p.checkoutErr
}

func (p *fakeProvider) ReconcileAttempt(_ context.Context, _ ports.ReconciliationRequest) (ports.ReconciliationResult, error) {
	p.reconcileCalls++
	return p.reconcileResult, p.reconcileErr
}

type fakeUOW struct {
	err   error
	calls int
}

func (u *fakeUOW) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	u.calls++
	if u.err != nil {
		return u.err
	}
	return fn(ctx)
}

type fixture struct {
	payments *paymentRepo
	attempts *attemptRepo
	provider *fakeProvider
	uow      *fakeUOW
	uc       *use_cases.InitiatePaymentUseCase
	input    use_cases.InitiatePaymentInput
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	amount, err := value_objects.NewMoney(25000, "NGN")
	if err != nil {
		t.Fatal(err)
	}

	payments := newPaymentRepo()
	attempts := newAttemptRepo()
	provider := &fakeProvider{
		checkoutResult: ports.CheckoutResult{
			ProviderReference: "provider-ref-001",
			CheckoutURL:       "https://payments.example.test/checkout/001",
			Status:            enums.PaymentAttemptStatusProcessing,
			Metadata:          json.RawMessage(`{"session":"test-session"}`),
		},
		reconcileResult: ports.ReconciliationResult{
			ProviderReference: "provider-ref-001",
			CheckoutURL:       "https://payments.example.test/checkout/001",
			Status:            enums.PaymentAttemptStatusProcessing,
		},
	}
	uow := &fakeUOW{}

	uc, err := use_cases.NewInitiatePaymentUseCase(
		payments, attempts, provider, uow, "test-provider",
	)
	if err != nil {
		t.Fatal(err)
	}

	return &fixture{
		payments: payments,
		attempts: attempts,
		provider: provider,
		uow:      uow,
		uc:       uc,
		input: use_cases.InitiatePaymentInput{
			PayerID:         testPayerID,
			Purpose:         enums.PaymentPurposeOrder,
			BusinessRefType: "order",
			BusinessRefID:   testOrderID,
			Amount:          amount,
			CustomerEmail:   "customer@example.com",
		},
	}
}

func TestInitiatePaymentCreatesPaymentAndAttempt(t *testing.T) {
	f := newFixture(t)

	got, err := f.uc.Execute(context.Background(), f.input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got == nil {
		t.Fatal("Execute() returned nil output")
	}
	if got.PaymentID == "" || got.AttemptID == "" {
		t.Fatal("expected payment and attempt UUIDs")
	}
	if err := application.ValidateUUID(got.PaymentID); err != nil {
		t.Errorf("payment ID is not a UUID: %v", err)
	}
	if err := application.ValidateUUID(got.AttemptID); err != nil {
		t.Errorf("attempt ID is not a UUID: %v", err)
	}
	if got.CheckoutURL != f.provider.checkoutResult.CheckoutURL {
		t.Errorf("checkout URL = %q", got.CheckoutURL)
	}
	if got.Provider != "test-provider" {
		t.Errorf("provider = %q", got.Provider)
	}
	if f.provider.checkoutCalls != 1 {
		t.Errorf("checkout calls = %d, want 1", f.provider.checkoutCalls)
	}
	if f.uow.calls != 2 {
		t.Errorf("transaction calls = %d, want 2", f.uow.calls)
	}
	if len(f.payments.items) != 1 || len(f.attempts.items) != 1 {
		t.Fatalf("expected one persisted payment and attempt; got %d and %d",
			len(f.payments.items), len(f.attempts.items))
	}

	payment, err := f.payments.GetByBusinessReference(
		context.Background(),
		paymentBusinessReference(t, f.input),
	)
	if err != nil {
		t.Fatal(err)
	}
	if payment.AttemptID() != got.AttemptID {
		t.Errorf("payment attempt link = %q, want %q", payment.AttemptID(), got.AttemptID)
	}
}

func TestInitiatePaymentReusesAndReconcilesExistingAttempt(t *testing.T) {
	f := newFixture(t)

	first, err := f.uc.Execute(context.Background(), f.input)
	if err != nil {
		t.Fatalf("first Execute() error = %v", err)
	}

	second, err := f.uc.Execute(context.Background(), f.input)
	if err != nil {
		t.Fatalf("second Execute() error = %v", err)
	}

	if second.PaymentID != first.PaymentID || second.AttemptID != first.AttemptID {
		t.Fatalf("idempotent retry created different IDs: first=%+v second=%+v", first, second)
	}
	if f.provider.checkoutCalls != 1 {
		t.Errorf("checkout calls = %d, want 1", f.provider.checkoutCalls)
	}
	if f.provider.reconcileCalls != 1 {
		t.Errorf("reconcile calls = %d, want 1", f.provider.reconcileCalls)
	}
}

func TestInitiatePaymentRejectsInvalidUUIDBeforeProviderCall(t *testing.T) {
	f := newFixture(t)
	f.input.PayerID = "not-a-uuid"

	_, err := f.uc.Execute(context.Background(), f.input)
	if err == nil {
		t.Fatal("expected invalid UUID error")
	}
	if f.provider.checkoutCalls != 0 {
		t.Errorf("provider called %d times for invalid input", f.provider.checkoutCalls)
	}
}

func TestInitiatePaymentPersistsAttemptWhenProviderReturnsError(t *testing.T) {
	f := newFixture(t)
	f.provider.checkoutErr = errors.New("provider unavailable")

	_, err := f.uc.Execute(context.Background(), f.input)
	if err == nil {
		t.Fatal("expected provider error")
	}
	if len(f.payments.items) != 1 || len(f.attempts.items) != 1 {
		t.Fatalf("expected durable payment and attempt after provider error; got %d and %d",
			len(f.payments.items), len(f.attempts.items))
	}
	if f.uow.calls != 2 {
		t.Errorf("transaction calls = %d, want 2", f.uow.calls)
	}
}

func TestInitiatePaymentReturnsPersistenceError(t *testing.T) {
	f := newFixture(t)
	f.uow.err = errors.New("database transaction failed")

	_, err := f.uc.Execute(context.Background(), f.input)
	if err == nil {
		t.Fatal("expected persistence error")
	}
	if f.uow.calls != 1 {
		t.Errorf("transaction calls = %d, want 1", f.uow.calls)
	}
}

func TestInitiatePaymentRejectsIdempotencyConflict(t *testing.T) {
	f := newFixture(t)

	if _, err := f.uc.Execute(context.Background(), f.input); err != nil {
		t.Fatalf("initial Execute() error = %v", err)
	}

	conflicting := f.input
	conflicting.Amount, _ = value_objects.NewMoney(30000, "NGN")

	_, err := f.uc.Execute(context.Background(), conflicting)
	if !errors.Is(err, application.ErrIdempotencyConflict) {
		t.Fatalf("error = %v, want ErrIdempotencyConflict", err)
	}
	if f.provider.checkoutCalls != 1 {
		t.Errorf("checkout calls = %d, want 1", f.provider.checkoutCalls)
	}
}

func paymentBusinessReference(t *testing.T, input use_cases.InitiatePaymentInput) value_objects.BusinessReference {
	t.Helper()
	ref, err := value_objects.NewBusinessReference(input.BusinessRefType, input.BusinessRefID)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}
