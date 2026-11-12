package infrastructure

import (
	"go.uber.org/fx"

	notification "github.com/iheanyi-dev/ecommerce-backend/services/notification"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/infrastructure/email"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/infrastructure/messaging/kafka"
)

var Module = fx.Module(
	"infrastructure",

	fx.Provide(
		notification.LoadConfig,

		fx.Annotate(
			email.NewSMTPSender,
			fx.As(new(ports.EmailSender)),
		),

		use_cases.NewSendEmailUseCase,
		kafka.NewEmailEventConsumer,
	),

	// Force construction so the consumer's lifecycle hooks are registered.
	fx.Invoke(func(*kafka.EmailEventConsumer) {}),
)
