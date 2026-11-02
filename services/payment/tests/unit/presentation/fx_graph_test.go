package presentation_test

import (
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/presentation"
	"go.uber.org/fx"
)

func TestPaymentFxDependencyGraph(t *testing.T) {
	err := fx.ValidateApp(
		infrastructure.Module,
		presentation.Module,
	)
	if err != nil {
		t.Fatalf("payment Fx dependency graph is invalid: %v", err)
	}
}
