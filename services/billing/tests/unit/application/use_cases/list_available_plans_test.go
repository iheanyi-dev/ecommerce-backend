package use_cases_test

import (
	"errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/value_objects"
)

type planCatalogStub struct {
	plans []entities.Plan
}

func (stub planCatalogStub) AvailablePlans() []entities.Plan {
	return stub.plans
}

func validPlan(planType, name string, amount int64) entities.Plan {
	price, _ := value_objects.NewMoney(amount, "NGN")

	return entities.Plan{
		ID:           uuid.New(),
		Type:         planType,
		Name:         name,
		Price:        price,
		Duration:     30 * 24 * time.Hour,
		Capabilities: entities.PlanCapabilities{MaxProducts: 5, AIEnabled: true},
		Available:    true,
	}
}

func TestListAvailablePlansReturnsAvailablePlans(t *testing.T) {
	premium := validPlan("premium", "Premium", 1200000)
	unavailable := validPlan("legacy", "Legacy", 500000)
	unavailable.Available = false

	useCase := use_cases.NewListAvailablePlans(planCatalogStub{
		plans: []entities.Plan{premium, unavailable},
	})

	got, err := useCase.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("Execute() returned %d plans, want 1", len(got))
	}

	if got[0].Type != "premium" {
		t.Errorf("plan type = %q, want premium", got[0].Type)
	}

	if got[0].Price.AmountMinor != 1200000 {
		t.Errorf("price = %d, want 1200000", got[0].Price.AmountMinor)
	}

	if got[0].Duration != 30*24*time.Hour {
		t.Errorf("duration = %v, want 30 days", got[0].Duration)
	}
}

func TestListAvailablePlansReturnsEmptyListWhenNoPlansAreAvailable(t *testing.T) {
	useCase := use_cases.NewListAvailablePlans(planCatalogStub{
		plans: []entities.Plan{},
	})

	got, err := useCase.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got == nil {
		t.Fatal("Execute() returned nil list, want empty list")
	}

	if len(got) != 0 {
		t.Fatalf("Execute() returned %d plans, want 0", len(got))
	}
}

func TestListAvailablePlansRejectsInvalidAvailablePlan(t *testing.T) {
	invalid := validPlan("", "Invalid", 1000)

	useCase := use_cases.NewListAvailablePlans(planCatalogStub{
		plans: []entities.Plan{invalid},
	})

	_, err := useCase.Execute()
	if !errors.Is(err, domain.ErrInvalidPlan) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidPlan)
	}
}

func TestListAvailablePlansRejectsMissingCatalog(t *testing.T) {
	useCase := use_cases.NewListAvailablePlans(nil)

	_, err := useCase.Execute()
	if !errors.Is(err, application.ErrPlanCatalogUnavailable) {
		t.Fatalf(
			"Execute() error = %v, want %v",
			err,
			application.ErrPlanCatalogUnavailable,
		)
	}
}
