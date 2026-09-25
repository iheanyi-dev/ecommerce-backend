package infrastructure_test

import (
	"testing"

	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure"
)

func TestModule_ProvidesInfrastructureDependencies(t *testing.T) {
	err := fx.ValidateApp(
		infrastructure.Module,
	)

	if err != nil {
		t.Fatalf("expected infrastructure module to validate, got: %v", err)
	}
}
