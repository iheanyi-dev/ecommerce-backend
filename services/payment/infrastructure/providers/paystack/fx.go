package paystack

import (
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/shared/config"
)

// Module wires the Paystack adapters to the Payment application ports.
// All adapters share one configured HTTP client.
var Module = fx.Module(
	"payment-paystack-provider",

	fx.Provide(
		newConfiguredClient,

		fx.Annotate(
			NewPaymentProvider,
			fx.As(new(ports.PaymentProvider)),
		),
		fx.Annotate(
			NewPayoutProvider,
			fx.As(new(ports.PayoutProvider)),
		),
		fx.Annotate(
			NewWebhookProvider,
			fx.As(new(ports.WebhookProvider)),
		),
	),
)

func newConfiguredClient(cfg *config.Config) (*Client, error) {
	return NewClient(cfg.PaystackSecretKey, cfg.PaystackBaseURL, nil)
}
