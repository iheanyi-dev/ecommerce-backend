package use_cases

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
)

type GetPayoutUseCase struct {
	payouts ports.PayoutRepository
}

func NewGetPayoutUseCase(payouts ports.PayoutRepository) (*GetPayoutUseCase, error) {
	if payouts == nil {
		return nil, fmt.Errorf("payout repository is required")
	}
	return &GetPayoutUseCase{payouts: payouts}, nil
}

func (uc *GetPayoutUseCase) Execute(
	ctx context.Context,
	payoutID string,
	storeID string,
) (*entities.Payout, error) {
	payoutID = strings.TrimSpace(payoutID)
	storeID = strings.TrimSpace(storeID)
	if payoutID == "" || storeID == "" {
		return nil, fmt.Errorf("payout ID and store ID are required")
	}

	payout, err := uc.payouts.GetByID(ctx, payoutID)
	if err != nil {
		if errors.Is(err, application.ErrPayoutNotFound) {
			return nil, application.ErrPayoutNotFound
		}
		return nil, fmt.Errorf("get payout: %w", err)
	}

	// Do not disclose whether a payout belongs to another store.
	if payout.StoreID() != storeID {
		return nil, application.ErrPayoutNotFound
	}
	return payout, nil
}
