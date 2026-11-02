package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/use_cases"
)

const (
	testOrderID2 = "33333333-3333-4333-8333-333333333333"
	testOrderID3 = "44444444-4444-4444-8444-444444444444"
	testPayerID2 = "55555555-5555-4555-8555-555555555555"
)

func TestListPaymentsByPayerIDReturnsPaginatedResults(t *testing.T) {
	f := newFixture(t)

	for _, id := range []string{testOrderID, testOrderID2, testOrderID3} {
		input := f.input
		input.BusinessRefID = id
		if _, err := f.uc.Execute(context.Background(), input); err != nil {
			t.Fatalf("create test payment: %v", err)
		}
	}

	uc, err := use_cases.NewListPaymentsByPayerIDUseCase(f.payments, f.attempts)
	if err != nil {
		t.Fatal(err)
	}

	first, err := uc.Execute(context.Background(), use_cases.ListPaymentsByPayerIDInput{
		PayerID: testPayerID, Page: 1, PageSize: 2,
	})
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if len(first.Items) != 2 || first.Total != 3 || first.TotalPages != 2 {
		t.Fatalf("unexpected first page: items=%d total=%d pages=%d",
			len(first.Items), first.Total, first.TotalPages)
	}
	for _, item := range first.Items {
		if item.PayerID != testPayerID {
			t.Fatalf("returned payment for wrong payer %q", item.PayerID)
		}
		if item.Attempt == nil {
			t.Fatal("expected attempt details for each payment")
		}
	}

	second, err := uc.Execute(context.Background(), use_cases.ListPaymentsByPayerIDInput{
		PayerID: testPayerID, Page: 2, PageSize: 2,
	})
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if len(second.Items) != 1 || second.Total != 3 || second.TotalPages != 2 {
		t.Fatalf("unexpected second page: items=%d total=%d pages=%d",
			len(second.Items), second.Total, second.TotalPages)
	}
}

func TestListPaymentsByPayerIDReturnsEmptyPage(t *testing.T) {
	f := newFixture(t)
	uc, err := use_cases.NewListPaymentsByPayerIDUseCase(f.payments, f.attempts)
	if err != nil {
		t.Fatal(err)
	}

	got, err := uc.Execute(context.Background(), use_cases.ListPaymentsByPayerIDInput{
		PayerID: testPayerID2,
	})
	if err != nil {
		t.Fatalf("Execute(): %v", err)
	}
	if len(got.Items) != 0 || got.Total != 0 || got.TotalPages != 0 {
		t.Fatalf("expected empty result, got %+v", got)
	}
	if got.Page != 1 || got.PageSize != use_cases.DefaultPaymentPageSize {
		t.Fatalf("unexpected defaults: page=%d size=%d", got.Page, got.PageSize)
	}
}

func TestListPaymentsByPayerIDRejectsInvalidUUID(t *testing.T) {
	f := newFixture(t)
	uc, err := use_cases.NewListPaymentsByPayerIDUseCase(f.payments, f.attempts)
	if err != nil {
		t.Fatal(err)
	}
	_, err = uc.Execute(context.Background(), use_cases.ListPaymentsByPayerIDInput{
		PayerID: "not-a-uuid",
	})
	if err == nil {
		t.Fatal("expected UUID validation error")
	}
}

func TestListPaymentsByPayerIDRejectsInvalidPagination(t *testing.T) {
	f := newFixture(t)
	uc, err := use_cases.NewListPaymentsByPayerIDUseCase(f.payments, f.attempts)
	if err != nil {
		t.Fatal(err)
	}

	cases := []use_cases.ListPaymentsByPayerIDInput{
		{PayerID: testPayerID, Page: -1, PageSize: 10},
		{PayerID: testPayerID, Page: 1, PageSize: -1},
		{PayerID: testPayerID, Page: 1, PageSize: use_cases.MaxPaymentPageSize + 1},
		{PayerID: testPayerID, Page: int(^uint(0) >> 1), PageSize: 2},
	}
	for _, input := range cases {
		_, err := uc.Execute(context.Background(), input)
		if !errors.Is(err, application.ErrInvalidPagination) {
			t.Errorf("input %+v: error = %v, want ErrInvalidPagination", input, err)
		}
	}
}

func TestListPaymentsByPayerIDDoesNotReturnAnotherPayersPayments(t *testing.T) {
	f := newFixture(t)

	other := f.input
	other.PayerID = testPayerID2
	if _, err := f.uc.Execute(context.Background(), other); err != nil {
		t.Fatalf("create other payer payment: %v", err)
	}

	uc, err := use_cases.NewListPaymentsByPayerIDUseCase(f.payments, f.attempts)
	if err != nil {
		t.Fatal(err)
	}
	got, err := uc.Execute(context.Background(), use_cases.ListPaymentsByPayerIDInput{
		PayerID: testPayerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 0 {
		t.Fatalf("returned %d payments belonging to another payer", len(got.Items))
	}
}
