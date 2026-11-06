package application

import (
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/use_cases"
)

// Module wires Order application use cases.
var Module = fx.Module(
	"order-application",
	fx.Provide(
		use_cases.NewCreateOrder,
		use_cases.NewGetOrder,
		use_cases.NewListOrders,
		use_cases.NewAddShipping,
		use_cases.NewRejectShipping,
		use_cases.NewInitiatePayment,
		use_cases.NewInitiatePayout,
		use_cases.NewHandlePayment,
		use_cases.NewHandlePayout,
		use_cases.NewPaymentEventHandler,
		use_cases.NewCancelOrder,
		use_cases.NewMarkSent,
		use_cases.NewConfirmReceipt,
	),
)
