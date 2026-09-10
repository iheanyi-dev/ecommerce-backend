package observability_test

import (
	"testing"

	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/shared"
)

func TestInfrastructureModule_ValidatesObservabilityDependencies(
	t *testing.T,
) {
	err := fx.ValidateApp(
		shared.Module,
		infrastructure.Module,
	)

	if err != nil {
		t.Fatalf(
			"expected infrastructure Fx graph to validate successfully, got %v",
			err,
		)
	}
}
