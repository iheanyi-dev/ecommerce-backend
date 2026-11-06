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

type payoutRepoFake struct {
	items   map[string]*entities.Payout
	saveErr error
}

func newPayoutRepoFake() *payoutRepoFake {
	return &payoutRepoFake{items: make(map[string]*entities.Payout)}
}

func payoutKey(ref value_objects.BusinessReference) string {
	return ref.Type() + ":" + ref.ID()
}

func (r *payoutRepoFake) GetByID(_ context.Context, id string) (*entities.Payout, error) {
	for _, p := range r.items {
		if p.ID() == id {
			return p, nil
		}
	}
	return nil, application.ErrPayoutNotFound
}

func (r *payoutRepoFake) GetByBusinessReference(_ context.Context, ref value_objects.BusinessReference) (*entities.Payout, error) {
	p, ok := r.items[payoutKey(ref)]
	if !ok {
		return nil, application.ErrPayoutNotFound
	}
	return p, nil
}

func (r *payoutRepoFake) ListByStoreID(_ context.Context, storeID string, limit, offset int) ([]*entities.Payout, int64, error) {
	var result []*entities.Payout
	for _, p := range r.items {
		if p.StoreID() == storeID {
			result = append(result, p)
		}
	}
	total := int64(len(result))
	if offset >= len(result) {
		return []*entities.Payout{}, total, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], total, nil
}

func (r *payoutRepoFake) Save(_ context.Context, p *entities.Payout) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.items[payoutKey(p.BusinessReference())] = p
	return nil
}

type payoutAttemptRepoFake struct {
	items   map[string]*entities.PayoutAttempt
	saveErr error
}

func newPayoutAttemptRepoFake() *payoutAttemptRepoFake {
	return &payoutAttemptRepoFake{items: make(map[string]*entities.PayoutAttempt)}
}

func (r *payoutAttemptRepoFake) GetByID(_ context.Context, id string) (*entities.PayoutAttempt, error) {
	a, ok := r.items[id]
	if !ok {
		return nil, application.ErrPayoutAttemptNotFound
	}
	return a, nil
}

func (r *payoutAttemptRepoFake) GetByPayoutID(_ context.Context, payoutID string) (*entities.PayoutAttempt, error) {
	for _, a := range r.items {
		if a.PayoutID() == payoutID {
			return a, nil
		}
	}
	return nil, application.ErrPayoutAttemptNotFound
}

func (r *payoutAttemptRepoFake) Save(_ context.Context, a *entities.PayoutAttempt) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.items[a.ID()] = a
	return nil
}

type payoutProviderFake struct {
	initResult      ports.PayoutInitiationResult
	initErr         error
	reconcileResult ports.PayoutReconciliationResult
	reconcileErr    error
	initCalls       int
	reconcileCalls  int
}

func (p *payoutProviderFake) InitiatePayout(_ context.Context, _ ports.PayoutInitiationRequest) (ports.PayoutInitiationResult, error) {
	p.initCalls++
	return p.initResult, p.initErr
}

func (p *payoutProviderFake) ReconcilePayout(_ context.Context, _ ports.PayoutReconciliationRequest) (ports.PayoutReconciliationResult, error) {
	p.reconcileCalls++
	return p.reconcileResult, p.reconcileErr
}

type payoutFixture struct {
	payments *paymentRepo
	payouts  *payoutRepoFake
	attempts *payoutAttemptRepoFake
	provider *payoutProviderFake
	outbox   *fakeOutbox
	uow      *fakeUOW
	uc       *use_cases.InitiatePayoutUseCase
	input    ports.PayoutRequest
}

func newPayoutFixture(t *testing.T) *payoutFixture {
	t.Helper()

	ref, err := value_objects.NewBusinessReference("order", testOrderID)
	if err != nil {
		t.Fatal(err)
	}
	paidAmount, err := value_objects.NewMoney(25000, "NGN")
	if err != nil {
		t.Fatal(err)
	}
	payoutAmount, err := value_objects.NewMoney(9500, "NGN")
	if err != nil {
		t.Fatal(err)
	}

	payments := newPaymentRepo()
	payment, err := entities.NewPayment(testPayerID, enums.PaymentPurposeOrder, ref, paidAmount)
	if err != nil {
		t.Fatal(err)
	}
	if err := payment.MarkProcessing(); err != nil {
		t.Fatal(err)
	}
	if err := payment.MarkSucceeded(); err != nil {
		t.Fatal(err)
	}
	if err := payments.Save(context.Background(), payment); err != nil {
		t.Fatal(err)
	}

	payouts := newPayoutRepoFake()
	attempts := newPayoutAttemptRepoFake()
	provider := &payoutProviderFake{
		initResult: ports.PayoutInitiationResult{
			ProviderReference: "transfer-ref-001",
			Status:            enums.PayoutAttemptStatusProcessing,
			Metadata:          json.RawMessage(`{"transfer":"test"}`),
		},
		reconcileResult: ports.PayoutReconciliationResult{
			Amount:            payoutAmount.Amount(),
			Currency:          payoutAmount.Currency(),
			ProviderReference: "transfer-ref-001",
			Status:            enums.PayoutAttemptStatusProcessing,
		},
	}
	outbox := &fakeOutbox{}
	uow := &fakeUOW{}
	uc, err := use_cases.NewInitiatePayoutUseCase(
		payments, payouts, attempts, provider, outbox, uow, "test-provider",
	)
	if err != nil {
		t.Fatal(err)
	}

	return &payoutFixture{
		payments: payments,
		payouts:  payouts,
		attempts: attempts,
		provider: provider,
		outbox:   outbox,
		uow:      uow,
		uc:       uc,
		input: ports.PayoutRequest{
			BusinessReference: ref,
			StoreID:           "store-123",
			OrderID:           testOrderID,
			Reason:            "Order fulfillment",
			Amount:            payoutAmount,
			Beneficiary: entities.PayoutBeneficiary{
				AccountName:   "Test Merchant",
				AccountNumber: "0123456789",
				BankCode:      "000001",
				BankName:      "Test Bank",
				CountryCode:   "NG",
			},
		},
	}
}

func TestInitiatePayoutCreatesPayoutAndAttempt(t *testing.T) {
	f := newPayoutFixture(t)

	got, err := f.uc.Execute(context.Background(), f.input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.PayoutID == "" || got.AttemptID == "" {
		t.Fatal("expected payout and attempt IDs")
	}
	if err := application.ValidateUUID(got.PayoutID); err != nil {
		t.Errorf("payout ID is not a UUID: %v", err)
	}
	if err := application.ValidateUUID(got.AttemptID); err != nil {
		t.Errorf("attempt ID is not a UUID: %v", err)
	}
	if got.Status != enums.PayoutStatusProcessing {
		t.Errorf("payout status = %q", got.Status)
	}
	if got.AttemptStatus != enums.PayoutAttemptStatusProcessing {
		t.Errorf("attempt status = %q", got.AttemptStatus)
	}
	if f.provider.initCalls != 1 || f.uow.calls != 2 {
		t.Fatalf("provider calls=%d transaction calls=%d; want 1 and 2", f.provider.initCalls, f.uow.calls)
	}
	if len(f.payouts.items) != 1 || len(f.attempts.items) != 1 {
		t.Fatalf("persisted payouts=%d attempts=%d", len(f.payouts.items), len(f.attempts.items))
	}
}

func TestInitiatePayoutRejectsUnpaidOrder(t *testing.T) {
	f := newPayoutFixture(t)
	payment, err := f.payments.GetByBusinessReference(context.Background(), f.input.BusinessReference)
	if err != nil {
		t.Fatal(err)
	}
	// Replace the paid payment with a fresh pending payment for the same reference.
	pending, err := entities.NewPayment(testPayerID, enums.PaymentPurposeOrder, f.input.BusinessReference, payment.Amount())
	if err != nil {
		t.Fatal(err)
	}
	if err := f.payments.Save(context.Background(), pending); err != nil {
		t.Fatal(err)
	}

	_, err = f.uc.Execute(context.Background(), f.input)
	if err == nil {
		t.Fatal("expected unpaid payment to be rejected")
	}
	if f.provider.initCalls != 0 {
		t.Fatalf("provider called %d times for an unpaid order", f.provider.initCalls)
	}
	if len(f.payouts.items) != 0 {
		t.Fatal("payout persisted for an unpaid order")
	}
}

func TestInitiatePayoutIsIdempotentAndReconcilesExistingAttempt(t *testing.T) {
	f := newPayoutFixture(t)

	first, err := f.uc.Execute(context.Background(), f.input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := f.uc.Execute(context.Background(), f.input)
	if err != nil {
		t.Fatal(err)
	}
	if first.PayoutID != second.PayoutID || first.AttemptID != second.AttemptID {
		t.Fatalf("payout/attempt changed across duplicate requests: first=%+v second=%+v", first, second)
	}
	if f.provider.initCalls != 1 || f.provider.reconcileCalls != 1 {
		t.Fatalf("init calls=%d reconcile calls=%d", f.provider.initCalls, f.provider.reconcileCalls)
	}
}

func TestInitiatePayoutRejectsIdempotencyConflict(t *testing.T) {
	f := newPayoutFixture(t)
	if _, err := f.uc.Execute(context.Background(), f.input); err != nil {
		t.Fatal(err)
	}

	conflicting := f.input
	conflicting.Reason = "Different reason"
	_, err := f.uc.Execute(context.Background(), conflicting)
	if !errors.Is(err, application.ErrPayoutIdempotencyConflict) {
		t.Fatalf("error = %v, want idempotency conflict", err)
	}
	if f.provider.initCalls != 1 {
		t.Fatalf("provider initiation calls = %d, want 1", f.provider.initCalls)
	}
}

func TestInitiatePayoutDoesNotRetryFailedAttemptImplicitly(t *testing.T) {
	f := newPayoutFixture(t)
	f.provider.initResult.Status = enums.PayoutAttemptStatusProcessing
	first, err := f.uc.Execute(context.Background(), f.input)
	if err != nil {
		t.Fatal(err)
	}

	attempt, err := f.attempts.GetByID(context.Background(), first.AttemptID)
	if err != nil {
		t.Fatal(err)
	}
	if err := attempt.MarkFailed("bank rejected transfer"); err != nil {
		t.Fatal(err)
	}
	payout, err := f.payouts.GetByBusinessReference(context.Background(), f.input.BusinessReference)
	if err != nil {
		t.Fatal(err)
	}
	if err := payout.MarkFailed(); err != nil {
		t.Fatal(err)
	}

	second, err := f.uc.Execute(context.Background(), f.input)
	if err != nil {
		t.Fatal(err)
	}
	if second.AttemptID != first.AttemptID {
		t.Fatalf("implicit retry replaced attempt %q with %q", first.AttemptID, second.AttemptID)
	}
	if f.provider.initCalls != 1 {
		t.Fatalf("provider initiation calls = %d, want 1", f.provider.initCalls)
	}
}

func TestRetryPayoutRequiresExplicitRetryOfDefinitivelyFailedAttempt(t *testing.T) {
	f := newPayoutFixture(t)
	first, err := f.uc.Execute(context.Background(), f.input)
	if err != nil {
		t.Fatal(err)
	}

	oldAttempt, err := f.attempts.GetByID(context.Background(), first.AttemptID)
	if err != nil {
		t.Fatal(err)
	}
	if err := oldAttempt.MarkFailed("bank rejected transfer"); err != nil {
		t.Fatal(err)
	}
	payout, err := f.payouts.GetByBusinessReference(context.Background(), f.input.BusinessReference)
	if err != nil {
		t.Fatal(err)
	}
	if err := payout.MarkFailed(); err != nil {
		t.Fatal(err)
	}

	retry, err := use_cases.NewRetryPayoutUseCase(
		f.payouts, f.attempts, f.provider, f.outbox, f.uow, "test-provider",
	)
	if err != nil {
		t.Fatal(err)
	}
	f.provider.initResult.Status = enums.PayoutAttemptStatusProcessing

	got, err := retry.Execute(context.Background(), first.PayoutID, f.input.StoreID)
	if err != nil {
		t.Fatalf("retry failed: %v", err)
	}
	if got.AttemptID == first.AttemptID {
		t.Fatal("explicit retry must create a new attempt")
	}
	if got.Status != enums.PayoutStatusProcessing {
		t.Fatalf("payout status = %q", got.Status)
	}
	if len(f.attempts.items) != 2 {
		t.Fatalf("attempt count = %d, want 2", len(f.attempts.items))
	}
	if f.provider.initCalls != 2 {
		t.Fatalf("provider initiation calls = %d, want 2", f.provider.initCalls)
	}
}

func TestRetryPayoutRejectsNonFailedAttempt(t *testing.T) {
	f := newPayoutFixture(t)
	first, err := f.uc.Execute(context.Background(), f.input)
	if err != nil {
		t.Fatal(err)
	}
	retry, err := use_cases.NewRetryPayoutUseCase(
		f.payouts, f.attempts, f.provider, f.outbox, f.uow, "test-provider",
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = retry.Execute(context.Background(), first.PayoutID, f.input.StoreID)
	if !errors.Is(err, application.ErrPayoutRetryNotAllowed) {
		t.Fatalf("error = %v, want retry-not-allowed", err)
	}
	if f.provider.initCalls != 1 {
		t.Fatalf("provider initiation calls = %d, want 1", f.provider.initCalls)
	}
}

func TestGetPayoutEnforcesStoreScope(t *testing.T) {
	f := newPayoutFixture(t)
	created, err := f.uc.Execute(context.Background(), f.input)
	if err != nil {
		t.Fatal(err)
	}
	get, err := use_cases.NewGetPayoutUseCase(f.payouts)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := get.Execute(context.Background(), created.PayoutID, "another-store"); !errors.Is(err, application.ErrPayoutNotFound) {
		t.Fatalf("cross-store error = %v, want payout not found", err)
	}
	if _, err := get.Execute(context.Background(), created.PayoutID, f.input.StoreID); err != nil {
		t.Fatalf("get payout failed: %v", err)
	}
}

func TestListPayoutsByStoreIDValidatesScopeAndPagination(t *testing.T) {
	f := newPayoutFixture(t)
	if _, err := f.uc.Execute(context.Background(), f.input); err != nil {
		t.Fatal(err)
	}
	list, err := use_cases.NewListPayoutsByStoreIDUseCase(f.payouts)
	if err != nil {
		t.Fatal(err)
	}

	got, err := list.Execute(context.Background(), f.input.StoreID, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 1 || len(got.Payouts) != 1 {
		t.Fatalf("total=%d payouts=%d, want 1 and 1", got.Total, len(got.Payouts))
	}
	if _, err := list.Execute(context.Background(), f.input.StoreID, 0, 0); !errors.Is(err, application.ErrInvalidPagination) {
		t.Fatalf("invalid limit error = %v", err)
	}
	if _, err := list.Execute(context.Background(), "", 10, 0); !errors.Is(err, application.ErrInvalidPagination) {
		t.Fatalf("empty store error = %v", err)
	}
}

func TestInitiatePayoutEmitsEventWhenImmediatelySucceeded(t *testing.T) {
	f := newPayoutFixture(t)
	f.provider.initResult.Status = enums.PayoutAttemptStatusSucceeded

	got, err := f.uc.Execute(context.Background(), f.input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.Status != enums.PayoutStatusSucceeded {
		t.Fatalf("payout status = %q, want succeeded", got.Status)
	}
	if len(f.outbox.events) != 1 {
		t.Fatalf("outbox events = %d, want 1", len(f.outbox.events))
	}

	var payload struct {
		PayoutID  string `json:"payout_id"`
		AttemptID string `json:"attempt_id"`
		Status    string `json:"status"`
	}
	if err := json.Unmarshal(f.outbox.events[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.PayoutID != got.PayoutID ||
		payload.AttemptID != got.AttemptID ||
		payload.Status != string(enums.PayoutStatusSucceeded) {
		t.Fatalf("unexpected payout event payload: %+v", payload)
	}
}

func TestInitiatePayoutDoesNotEmitEventWhileProcessing(t *testing.T) {
	f := newPayoutFixture(t)

	if _, err := f.uc.Execute(context.Background(), f.input); err != nil {
		t.Fatal(err)
	}
	if len(f.outbox.events) != 0 {
		t.Fatalf("outbox events = %d, want 0 for processing payout", len(f.outbox.events))
	}
}

func TestRetryPayoutEmitsEventForNewSuccessfulAttempt(t *testing.T) {
	f := newPayoutFixture(t)
	first, err := f.uc.Execute(context.Background(), f.input)
	if err != nil {
		t.Fatal(err)
	}

	oldAttempt, err := f.attempts.GetByID(context.Background(), first.AttemptID)
	if err != nil {
		t.Fatal(err)
	}
	if err := oldAttempt.MarkFailed("bank rejected transfer"); err != nil {
		t.Fatal(err)
	}
	payout, err := f.payouts.GetByBusinessReference(context.Background(), f.input.BusinessReference)
	if err != nil {
		t.Fatal(err)
	}
	if err := payout.MarkFailed(); err != nil {
		t.Fatal(err)
	}

	f.provider.initResult.Status = enums.PayoutAttemptStatusSucceeded
	retry, err := use_cases.NewRetryPayoutUseCase(
		f.payouts, f.attempts, f.provider, f.outbox, f.uow, "test-provider",
	)
	if err != nil {
		t.Fatal(err)
	}

	got, err := retry.Execute(context.Background(), first.PayoutID, f.input.StoreID)
	if err != nil {
		t.Fatalf("retry Execute() error = %v", err)
	}
	if got.Status != enums.PayoutStatusSucceeded {
		t.Fatalf("payout status = %q, want succeeded", got.Status)
	}
	if got.AttemptID == first.AttemptID {
		t.Fatal("retry reused the previous attempt ID")
	}
	if len(f.outbox.events) != 1 {
		t.Fatalf("outbox events = %d, want 1", len(f.outbox.events))
	}

	var payload struct {
		AttemptID string `json:"attempt_id"`
		Status    string `json:"status"`
	}
	if err := json.Unmarshal(f.outbox.events[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.AttemptID != got.AttemptID ||
		payload.Status != string(enums.PayoutStatusSucceeded) {
		t.Fatalf("unexpected retry event payload: %+v", payload)
	}
}
