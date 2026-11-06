package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/handlers"
)

type initiatePayoutMock struct {
	called      bool
	orderID     uuid.UUID
	beneficiary ports.PayoutBeneficiary
	output      *dto.InitiatePayoutOutput
	err         error
}

func (m *initiatePayoutMock) Execute(
	_ context.Context,
	orderID uuid.UUID,
	beneficiary ports.PayoutBeneficiary,
) (*dto.InitiatePayoutOutput, error) {
	m.called = true
	m.orderID = orderID
	m.beneficiary = beneficiary
	return m.output, m.err
}

var _ ports.InitiatePayoutService = (*initiatePayoutMock)(nil)

func TestInitiatePayoutHandler_Success(t *testing.T) {
	orderID := uuid.New()
	payoutID := uuid.New()
	attemptID := uuid.New()

	service := &initiatePayoutMock{
		output: &dto.InitiatePayoutOutput{
			PayoutID:      payoutID,
			AttemptID:     attemptID,
			Reference:     "payout-reference",
			Status:        "pending",
			AttemptStatus: "processing",
			Provider:      "paystack",
			AmountMinor:   95000,
			Currency:      "NGN",
		},
	}

	handler := handlers.NewInitiatePayoutHandler(service)
	body := `{
		"order_id": "` + orderID.String() + `",
		"account_name": " Test Account ",
		"account_number": " 0123456789 ",
		"bank_code": " 001 ",
		"bank_name": " Test Bank ",
		"country_code": " NG "
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/stores/owner/payouts",
		bytes.NewBufferString(body),
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.True(t, service.called)
	require.Equal(t, orderID, service.orderID)
	require.Equal(t, ports.PayoutBeneficiary{
		AccountName:   "Test Account",
		AccountNumber: "0123456789",
		BankCode:      "001",
		BankName:      "Test Bank",
		CountryCode:   "NG",
	}, service.beneficiary)

	var response struct {
		PayoutID      string `json:"payout_id"`
		AttemptID     string `json:"attempt_id"`
		Reference     string `json:"reference"`
		Status        string `json:"status"`
		AttemptStatus string `json:"attempt_status"`
		Provider      string `json:"provider"`
		AmountMinor   int64  `json:"amount_minor"`
		Currency      string `json:"currency"`
	}

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, payoutID.String(), response.PayoutID)
	require.Equal(t, attemptID.String(), response.AttemptID)
	require.Equal(t, "payout-reference", response.Reference)
	require.Equal(t, "pending", response.Status)
	require.Equal(t, "processing", response.AttemptStatus)
	require.Equal(t, "paystack", response.Provider)
	require.Equal(t, int64(95000), response.AmountMinor)
	require.Equal(t, "NGN", response.Currency)
}

func TestInitiatePayoutHandler_RejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "malformed JSON",
			body: `{"order_id":`,
		},
		{
			name: "invalid order ID",
			body: `{
				"order_id": "not-a-uuid",
				"account_name": "Test",
				"account_number": "0123456789",
				"bank_code": "001",
				"bank_name": "Test Bank",
				"country_code": "NG"
			}`,
		},
		{
			name: "missing beneficiary field",
			body: `{
				"order_id": "11111111-1111-1111-1111-111111111111",
				"account_name": "Test",
				"account_number": "0123456789",
				"bank_code": "001",
				"bank_name": "Test Bank"
			}`,
		},
		{
			name: "blank beneficiary field",
			body: `{
				"order_id": "11111111-1111-1111-1111-111111111111",
				"account_name": "   ",
				"account_number": "0123456789",
				"bank_code": "001",
				"bank_name": "Test Bank",
				"country_code": "NG"
			}`,
		},
		{
			name: "trailing JSON value",
			body: `{
				"order_id": "11111111-1111-1111-1111-111111111111",
				"account_name": "Test",
				"account_number": "0123456789",
				"bank_code": "001",
				"bank_name": "Test Bank",
				"country_code": "NG"
			} {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &initiatePayoutMock{}
			handler := handlers.NewInitiatePayoutHandler(service)

			request := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/stores/owner/payouts",
				bytes.NewBufferString(tt.body),
			)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.False(t, service.called)
		})
	}
}

func TestInitiatePayoutHandler_RejectsUnsupportedMethod(t *testing.T) {
	service := &initiatePayoutMock{}
	handler := handlers.NewInitiatePayoutHandler(service)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/stores/owner/payouts",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusMethodNotAllowed, recorder.Code)
	require.False(t, service.called)
}

func TestInitiatePayoutHandler_MapsApplicationErrors(t *testing.T) {
	service := &initiatePayoutMock{
		err: stderrors.Join(
			application_errors.ErrInvalidPayoutRequest,
			stderrors.New("invalid beneficiary"),
		),
	}
	handler := handlers.NewInitiatePayoutHandler(service)

	body := `{
		"order_id": "11111111-1111-1111-1111-111111111111",
		"account_name": "Test",
		"account_number": "0123456789",
		"bank_code": "001",
		"bank_name": "Test Bank",
		"country_code": "NG"
	}`
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/stores/owner/payouts",
		bytes.NewBufferString(body),
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.True(t, service.called)
}
