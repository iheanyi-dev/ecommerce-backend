package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
)

// PayoutBeneficiary contains the bank details required to initiate a payout.
// Store does not persist these details.
type PayoutBeneficiary struct {
	AccountName   string
	AccountNumber string
	BankCode      string
	BankName      string
	CountryCode   string
}

// InitiatePayoutService initiates a payout for an order belonging to the
// authenticated store owner.
type InitiatePayoutService interface {
	Execute(
		ctx context.Context,
		orderID uuid.UUID,
		beneficiary PayoutBeneficiary,
	) (*dto.InitiatePayoutOutput, error)
}
