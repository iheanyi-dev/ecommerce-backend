package infrastructure_test

import (
	"net/http"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation"
	"go.uber.org/fx"
)

func TestModule_ProvidesCompleteStoreDependencyGraph(t *testing.T) {
	if err := fx.ValidateApp(
		infrastructure.Module,
		application.Module,
		presentation.Module,
		fx.Invoke(func(http.Handler) {}),
	); err != nil {
		t.Fatalf("Store dependency graph is incomplete: %v", err)
	}
}
