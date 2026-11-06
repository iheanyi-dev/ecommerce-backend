package main

import (
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/infrastructure"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/presentation"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		infrastructure.Module,
		presentation.Module,
	).Run()
}
