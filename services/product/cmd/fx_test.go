package main

import (
	"testing"

	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/infrastructure"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/presentation"
)

func TestProductModulesValidate(t *testing.T) {
	if err := fx.ValidateApp(
		infrastructure.Module,
		application.Module,
		presentation.Module,
	); err != nil {
		t.Fatalf("Product Fx modules failed validation: %v", err)
	}
}
