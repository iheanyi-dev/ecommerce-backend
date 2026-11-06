package use_cases_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/value_objects"
)

func validPlan() entities.Plan {
	price, _ := value_objects.NewMoney(5000, "NGN")

	return entities.Plan{
		ID:       uuid.New(),
		Type:     "premium",
		Name:     "Premium",
		Price:    price,
		Duration: 30 * 24 * time.Hour,
		Capabilities: entities.PlanCapabilities{
			MaxProducts: 5,
			AIEnabled:   true,
		},
		Available: true,
		CreatedAt: time.Now().UTC().Add(-24 * time.Hour),
		UpdatedAt: time.Now().UTC().Add(-24 * time.Hour),
	}
}

func validCreatePlanInput() use_cases.CreatePlanInput {
	price, _ := value_objects.NewMoney(5000, "NGN")

	return use_cases.CreatePlanInput{
		Type:        "premium",
		Name:        "Premium",
		Description: "Premium subscription",
		Price:       price,
		Duration:    30 * 24 * time.Hour,
		Capabilities: entities.PlanCapabilities{
			MaxProducts: 5,
			AIEnabled:   true,
		},
		Available: true,
	}
}

func validUpdatePlanInput() use_cases.UpdatePlanInput {
	input := validCreatePlanInput()

	return use_cases.UpdatePlanInput{
		Type:         input.Type,
		Name:         input.Name,
		Description:  input.Description,
		Price:        input.Price,
		Duration:     input.Duration,
		Capabilities: input.Capabilities,
		Available:    input.Available,
		Promotion:    input.Promotion,
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
