package observability

import "github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"

// NewBillingTracer provides the Billing application tracing port.
func NewBillingTracer() (ports.Tracer, error) {
	return NewOpenTelemetryTracer("billing")
}
