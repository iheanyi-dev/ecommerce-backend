package infrastructure_test

import (
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure"
	"go.uber.org/fx"
)

func TestModuleValidatesDependencyGraph(t *testing.T) {
	if err := fx.ValidateApp(infrastructure.Module); err != nil {
		t.Fatalf("Payment infrastructure module is invalid: %v", err)
	}
}
