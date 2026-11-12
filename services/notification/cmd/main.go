package main

import (
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/notification/infrastructure"
)

func main() {
	fx.New(
		infrastructure.Module,
	).Run()
}
