package postgres_test

import (
	"context"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure/persistence/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestNewUnitOfWorkRejectsNilPool(t *testing.T) {
	uow, err := postgres.NewUnitOfWork(nil)

	if err == nil {
		t.Fatal("expected an error for nil pool")
	}

	if uow != nil {
		t.Fatal("expected nil unit of work")
	}
}

func TestNewUnitOfWorkAcceptsPool(t *testing.T) {
	pool := &pgxpool.Pool{}

	uow, err := postgres.NewUnitOfWork(pool)

	if err != nil {
		t.Fatalf("create unit of work: %v", err)
	}

	if uow == nil {
		t.Fatal("expected a unit of work")
	}
}

func TestWithinTransactionRejectsNilCallback(t *testing.T) {
	pool := &pgxpool.Pool{}

	uow, err := postgres.NewUnitOfWork(pool)
	if err != nil {
		t.Fatalf("create unit of work: %v", err)
	}

	err = uow.WithinTransaction(context.Background(), nil)

	if err == nil {
		t.Fatal("expected an error for nil callback")
	}
}
