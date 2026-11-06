package kafka

import "go.uber.org/fx"

// Module provides Kafka event publishing and the payment outbox worker.
var Module = fx.Module(
	"payment-kafka",
	fx.Provide(
		fx.Annotate(
			NewProducer,
			fx.As(new(eventPublisher)),
		),
		NewOutboxWorker,
	),
)
