package main

import (
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/infrastructure"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/presentation"
	"go.uber.org/fx"
)

func TestCartModulesValidate(t *testing.T) {
	if err := fx.ValidateApp(
		infrastructure.Module,
		presentation.Module,
	); err != nil {
		t.Fatalf("Cart Fx modules are invalid: %v", err)
	}
}
