package main

import (
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/infrastructure"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/presentation"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		infrastructure.Module,
		presentation.Module,
	).Run()
}
