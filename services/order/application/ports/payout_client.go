package ports

import "context"

// PayoutClient initiates store payouts through the Payment service.
type PayoutClient interface {
	InitiatePayout(
		ctx context.Context,
		request InitiatePayoutRequest,
	) (*InitiatePayoutResponse, error)
}
