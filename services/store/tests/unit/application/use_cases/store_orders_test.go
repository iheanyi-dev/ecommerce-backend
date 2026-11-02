package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
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
	calls   int
	storeID uuid.UUID
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

	require.Equal(t, 5, owner.calls)
	require.Equal(t, 5, client.calls)

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
