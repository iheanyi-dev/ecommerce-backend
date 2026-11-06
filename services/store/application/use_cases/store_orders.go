package use_cases

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
)

type storeOrdersBase struct {
	owner  ports.GetStoreByOwnerService
	client ports.OrderClient
}

func (uc storeOrdersBase) ownedStore(ctx context.Context) (uuid.UUID, error) {
	store, err := uc.owner.Execute(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	if store == nil || store.ID.Value() == uuid.Nil {
		return uuid.Nil, applicationerrors.ErrStoreNotFound
	}
	return store.ID.Value(), nil
}

type ListStoreOrdersUseCase struct{ storeOrdersBase }

func NewListStoreOrdersUseCase(owner ports.GetStoreByOwnerService, client ports.OrderClient) *ListStoreOrdersUseCase {
	return &ListStoreOrdersUseCase{storeOrdersBase{owner: owner, client: client}}
}

func (uc *ListStoreOrdersUseCase) Execute(ctx context.Context, page, pageSize int) (*dto.ListStoreOrdersOutput, error) {
	if page < 1 || pageSize < 1 || pageSize > 100 {
		return nil, applicationerrors.ErrInvalidPagination
	}
	storeID, err := uc.ownedStore(ctx)
	if err != nil {
		return nil, err
	}
	return uc.client.ListStoreOrders(ctx, storeID, page, pageSize)
}

type GetStoreOrderUseCase struct{ storeOrdersBase }

func NewGetStoreOrderUseCase(owner ports.GetStoreByOwnerService, client ports.OrderClient) *GetStoreOrderUseCase {
	return &GetStoreOrderUseCase{storeOrdersBase{owner: owner, client: client}}
}

func (uc *GetStoreOrderUseCase) Execute(ctx context.Context, orderID uuid.UUID) (*dto.OrderOutput, error) {
	if orderID == uuid.Nil {
		return nil, fmt.Errorf("%w: order ID is required", applicationerrors.ErrInvalidPayoutRequest)
	}
	storeID, err := uc.ownedStore(ctx)
	if err != nil {
		return nil, err
	}
	return uc.client.GetStoreOrder(ctx, storeID, orderID)
}

type AddShippingUseCase struct{ storeOrdersBase }

func NewAddShippingUseCase(owner ports.GetStoreByOwnerService, client ports.OrderClient) *AddShippingUseCase {
	return &AddShippingUseCase{storeOrdersBase{owner: owner, client: client}}
}

func (uc *AddShippingUseCase) Execute(ctx context.Context, orderID uuid.UUID, shippingAmount int64) error {
	if orderID == uuid.Nil {
		return fmt.Errorf("order ID is required")
	}
	if shippingAmount < 0 {
		return fmt.Errorf("shipping amount cannot be negative")
	}
	storeID, err := uc.ownedStore(ctx)
	if err != nil {
		return err
	}
	return uc.client.AddShipping(ctx, storeID, orderID, shippingAmount)
}

type MarkOrderSentUseCase struct{ storeOrdersBase }

func NewMarkOrderSentUseCase(owner ports.GetStoreByOwnerService, client ports.OrderClient) *MarkOrderSentUseCase {
	return &MarkOrderSentUseCase{storeOrdersBase{owner: owner, client: client}}
}

func (uc *MarkOrderSentUseCase) Execute(ctx context.Context, orderID uuid.UUID) error {
	if orderID == uuid.Nil {
		return fmt.Errorf("order ID is required")
	}
	storeID, err := uc.ownedStore(ctx)
	if err != nil {
		return err
	}
	return uc.client.MarkSent(ctx, storeID, orderID)
}

type CancelStoreOrderUseCase struct{ storeOrdersBase }

func NewCancelStoreOrderUseCase(owner ports.GetStoreByOwnerService, client ports.OrderClient) *CancelStoreOrderUseCase {
	return &CancelStoreOrderUseCase{storeOrdersBase{owner: owner, client: client}}
}

func (uc *CancelStoreOrderUseCase) Execute(ctx context.Context, orderID uuid.UUID) error {
	if orderID == uuid.Nil {
		return fmt.Errorf("order ID is required")
	}
	storeID, err := uc.ownedStore(ctx)
	if err != nil {
		return err
	}
	return uc.client.CancelStoreOrder(ctx, storeID, orderID)
}

// InitiatePayoutUseCase initiates a payout for an order owned by the
// authenticated store owner.
type InitiatePayoutUseCase struct {
	storeOrdersBase
}

// NewInitiatePayoutUseCase creates the payout use case.
func NewInitiatePayoutUseCase(
	owner ports.GetStoreByOwnerService,
	client ports.OrderClient,
) *InitiatePayoutUseCase {
	return &InitiatePayoutUseCase{
		storeOrdersBase: storeOrdersBase{
			owner:  owner,
			client: client,
		},
	}
}

// Execute resolves the authenticated owner's store before requesting a payout.
func (uc *InitiatePayoutUseCase) Execute(
	ctx context.Context,
	orderID uuid.UUID,
	beneficiary ports.PayoutBeneficiary,
) (*dto.InitiatePayoutOutput, error) {
	if orderID == uuid.Nil {
		return nil, fmt.Errorf("%w: order ID is required", applicationerrors.ErrInvalidPayoutRequest)
	}

	if beneficiary.AccountName == "" ||
		beneficiary.AccountNumber == "" ||
		beneficiary.BankCode == "" ||
		beneficiary.BankName == "" ||
		beneficiary.CountryCode == "" {
		return nil, fmt.Errorf("%w: complete beneficiary details are required", applicationerrors.ErrInvalidPayoutRequest)
	}

	storeID, err := uc.ownedStore(ctx)
	if err != nil {
		return nil, err
	}

	return uc.client.InitiatePayout(ctx, storeID, orderID, beneficiary)
}
