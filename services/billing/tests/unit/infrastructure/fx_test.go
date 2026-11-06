package infrastructure_test

import (
	"testing"

	"go.uber.org/fx"

	billinginfra "github.com/iheanyi-dev/ecommerce-backend/services/billing/infrastructure"
	billingpresentation "github.com/iheanyi-dev/ecommerce-backend/services/billing/presentation"
)

func TestBillingModuleDependencyGraph(t *testing.T) {
	if err := fx.ValidateApp(
		billinginfra.Module,
		billingpresentation.Module,
	); err != nil {
		t.Fatalf("Billing Fx dependency graph is invalid: %v", err)
	}
}
