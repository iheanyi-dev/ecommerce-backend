package use_cases

import (
	"context"
	"fmt"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
)

type ListPayoutsByStoreIDOutput struct {
	Payouts []*entities.Payout
	Total   int64
	Limit   int
	Offset  int
}

type ListPayoutsByStoreIDUseCase struct {
	payouts ports.PayoutRepository
}

func NewListPayoutsByStoreIDUseCase(
	payouts ports.PayoutRepository,
) (*ListPayoutsByStoreIDUseCase, error) {
	if payouts == nil {
		return nil, fmt.Errorf("payout repository is required")
	}
	return &ListPayoutsByStoreIDUseCase{payouts: payouts}, nil
}

func (uc *ListPayoutsByStoreIDUseCase) Execute(
	ctx context.Context,
	storeID string,
	limit int,
	offset int,
) (*ListPayoutsByStoreIDOutput, error) {
	storeID = strings.TrimSpace(storeID)
	if storeID == "" || limit <= 0 || offset < 0 {
		return nil, application.ErrInvalidPagination
	}

	payouts, total, err := uc.payouts.ListByStoreID(ctx, storeID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list store payouts: %w", err)
	}
	if total < 0 || len(payouts) > limit {
		return nil, fmt.Errorf("payout repository returned invalid pagination data")
	}

	for _, payout := range payouts {
		if payout == nil || payout.StoreID() != storeID {
			return nil, fmt.Errorf("payout repository returned an invalid or out-of-scope payout")
		}
	}

	return &ListPayoutsByStoreIDOutput{
		Payouts: payouts,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}, nil
}
