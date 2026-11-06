package ports

import "context"

// PaymentEventHandler handles trusted payment and payout status events.
type PaymentEventHandler interface {
	HandlePayment(ctx context.Context, orderID, status string) error
	HandlePayout(ctx context.Context, orderID, status string) error
}
