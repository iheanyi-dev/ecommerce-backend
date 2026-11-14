package main

import (
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/infrastructure"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/presentation"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		infrastructure.Module,
		presentation.Module,
	).Run()
}
