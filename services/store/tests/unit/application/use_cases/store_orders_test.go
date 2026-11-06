package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
)

type orderOwnerStub struct {
	output *dto.StoreOutput
	err    error
	calls  int
}

func (s *orderOwnerStub) Execute(context.Context) (*dto.StoreOutput, error) {
	s.calls++
	return s.output, s.err
}

type orderClientStub struct {
	calls             int
	storeID           uuid.UUID
	payoutOrderID     uuid.UUID
	payoutBeneficiary ports.PayoutBeneficiary
	payoutOutput      *dto.InitiatePayoutOutput
}

func (s *orderClientStub) ListStoreOrders(_ context.Context, id uuid.UUID, page, size int) (*dto.ListStoreOrdersOutput, error) {
	s.calls++
	s.storeID = id
	return &dto.ListStoreOrdersOutput{Page: page, PageSize: size}, nil
}
func (s *orderClientStub) GetStoreOrder(_ context.Context, id, orderID uuid.UUID) (*dto.OrderOutput, error) {
	s.calls++
	s.storeID = id
	return &dto.OrderOutput{ID: orderID, StoreID: id}, nil
}
func (s *orderClientStub) AddShipping(_ context.Context, id, _ uuid.UUID, _ int64) error {
	s.calls++
	s.storeID = id
	return nil
}
func (s *orderClientStub) MarkSent(_ context.Context, id, _ uuid.UUID) error {
	s.calls++
	s.storeID = id
	return nil
}
func (s *orderClientStub) CancelStoreOrder(_ context.Context, id, _ uuid.UUID) error {
	s.calls++
	s.storeID = id
	return nil
}

func (s *orderClientStub) InitiatePayout(
	_ context.Context,
	storeID, orderID uuid.UUID,
	beneficiary ports.PayoutBeneficiary,
) (*dto.InitiatePayoutOutput, error) {
	s.calls++
	s.storeID = storeID
	s.payoutOrderID = orderID
	s.payoutBeneficiary = beneficiary
	if s.payoutOutput != nil {
		return s.payoutOutput, nil
	}
	return &dto.InitiatePayoutOutput{PayoutID: uuid.New(), AttemptID: uuid.New()}, nil
}
func TestStoreOrderUseCasesAuthorizeBeforeRPC(t *testing.T) {
	storeUUID := uuid.New()
	storeID, err := valueobjects.NewStoreID(storeUUID)
	require.NoError(t, err)
	owner := &orderOwnerStub{output: &dto.StoreOutput{ID: storeID}}
	client := &orderClientStub{}
	ctx := context.Background()

	list := use_cases.NewListStoreOrdersUseCase(owner, client)
	_, err = list.Execute(ctx, 1, 20)
	require.NoError(t, err)
	require.Equal(t, storeUUID, client.storeID)

	orderID := uuid.New()
	get := use_cases.NewGetStoreOrderUseCase(owner, client)
	_, err = get.Execute(ctx, orderID)
	require.NoError(t, err)

	add := use_cases.NewAddShippingUseCase(owner, client)
	require.NoError(t, add.Execute(ctx, orderID, 500))

	mark := use_cases.NewMarkOrderSentUseCase(owner, client)
	require.NoError(t, mark.Execute(ctx, orderID))

	cancel := use_cases.NewCancelStoreOrderUseCase(owner, client)
	require.NoError(t, cancel.Execute(ctx, orderID))

	payout := use_cases.NewInitiatePayoutUseCase(owner, client)
	payoutOutput, err := payout.Execute(ctx, orderID, ports.PayoutBeneficiary{
		AccountName:   "Store Owner",
		AccountNumber: "0123456789",
		BankCode:      "044",
		BankName:      "Access Bank",
		CountryCode:   "NG",
	})
	require.NoError(t, err)
	require.NotNil(t, payoutOutput)

	require.Equal(t, 6, owner.calls)
	require.Equal(t, 6, client.calls)

	owner.err = errors.New("owner lookup failed")
	before := client.calls
	_, err = get.Execute(ctx, orderID)
	require.Error(t, err)
	require.Equal(t, before, client.calls)
}

func TestListStoreOrdersRejectsInvalidPaginationBeforeRPC(t *testing.T) {
	owner := &orderOwnerStub{}
	client := &orderClientStub{}
	uc := use_cases.NewListStoreOrdersUseCase(owner, client)

	_, err := uc.Execute(context.Background(), 0, 20)
	require.ErrorIs(t, err, applicationerrors.ErrInvalidPagination)
	require.Zero(t, client.calls)
}

func TestInitiatePayoutResolvesAuthenticatedOwnersStore(t *testing.T) {
	storeUUID := uuid.New()
	orderID := uuid.New()
	storeID, err := valueobjects.NewStoreID(storeUUID)
	require.NoError(t, err)

	owner := &orderOwnerStub{output: &dto.StoreOutput{ID: storeID}}
	client := &orderClientStub{}
	beneficiary := ports.PayoutBeneficiary{
		AccountName:   "Test Store",
		AccountNumber: "0123456789",
		BankCode:      "044",
		BankName:      "Access Bank",
		CountryCode:   "NG",
	}
	expected := &dto.InitiatePayoutOutput{
		PayoutID:  uuid.New(),
		AttemptID: uuid.New(),
		Reference: "payout-reference",
		Status:    "pending",
	}
	client.payoutOutput = expected

	uc := use_cases.NewInitiatePayoutUseCase(owner, client)
	result, err := uc.Execute(context.Background(), orderID, beneficiary)

	require.NoError(t, err)
	require.Equal(t, expected, result)
	require.Equal(t, storeUUID, client.storeID)
	require.Equal(t, orderID, client.payoutOrderID)
	require.Equal(t, beneficiary, client.payoutBeneficiary)
	require.Equal(t, 1, owner.calls)
	require.Equal(t, 1, client.calls)
}

func TestInitiatePayoutRejectsInvalidRequestBeforeOwnerLookup(t *testing.T) {
	owner := &orderOwnerStub{}
	client := &orderClientStub{}
	uc := use_cases.NewInitiatePayoutUseCase(owner, client)

	_, err := uc.Execute(
		context.Background(),
		uuid.Nil,
		ports.PayoutBeneficiary{},
	)

	require.ErrorIs(t, err, applicationerrors.ErrInvalidPayoutRequest)
	require.Zero(t, owner.calls)
	require.Zero(t, client.calls)
}

func TestInitiatePayoutDoesNotCallOrderWhenOwnerLookupFails(t *testing.T) {
	owner := &orderOwnerStub{err: errors.New("owner lookup failed")}
	client := &orderClientStub{}
	uc := use_cases.NewInitiatePayoutUseCase(owner, client)

	_, err := uc.Execute(
		context.Background(),
		uuid.New(),
		ports.PayoutBeneficiary{
			AccountName:   "Test Store",
			AccountNumber: "0123456789",
			BankCode:      "044",
			BankName:      "Access Bank",
			CountryCode:   "NG",
		},
	)

	require.Error(t, err)
	require.Equal(t, 1, owner.calls)
	require.Zero(t, client.calls)
}
