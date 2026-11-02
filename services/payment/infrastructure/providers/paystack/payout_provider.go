package paystack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
)

type PayoutProvider struct {
	client *Client
}

func NewPayoutProvider(client *Client) (*PayoutProvider, error) {
	if client == nil {
		return nil, ErrInvalidConfiguration
	}
	return &PayoutProvider{client: client}, nil
}

func (p *PayoutProvider) InitiatePayout(
	ctx context.Context,
	request ports.PayoutInitiationRequest,
) (ports.PayoutInitiationResult, error) {
	if p == nil || p.client == nil {
		return ports.PayoutInitiationResult{}, ErrInvalidConfiguration
	}
	if request.Payout == nil || request.Attempt == nil ||
		request.Attempt.PayoutID() != request.Payout.ID() {
		return ports.PayoutInitiationResult{}, errors.New("invalid Paystack payout request")
	}

	payout := request.Payout
	beneficiary := payout.Beneficiary()

	if payout.Amount().IsZero() ||
		payout.Amount().Currency() != "NGN" ||
		strings.ToUpper(strings.TrimSpace(beneficiary.CountryCode)) != "NG" ||
		strings.TrimSpace(beneficiary.AccountName) == "" ||
		len(strings.TrimSpace(beneficiary.AccountNumber)) != 10 ||
		strings.TrimSpace(beneficiary.BankCode) == "" ||
		strings.TrimSpace(payout.Reason()) == "" {
		return ports.PayoutInitiationResult{}, errors.New(
			"Paystack payouts currently require a valid Nigerian NGN bank beneficiary",
		)
	}

	recipientPayload := struct {
		Type          string `json:"type"`
		Name          string `json:"name"`
		AccountNumber string `json:"account_number"`
		BankCode      string `json:"bank_code"`
		Currency      string `json:"currency"`
	}{
		Type:          "nuban",
		Name:          strings.TrimSpace(beneficiary.AccountName),
		AccountNumber: strings.TrimSpace(beneficiary.AccountNumber),
		BankCode:      strings.TrimSpace(beneficiary.BankCode),
		Currency:      "NGN",
	}

	var recipientResponse struct {
		RecipientCode string `json:"recipient_code"`
	}
	if err := p.client.Request(
		ctx,
		http.MethodPost,
		"/transferrecipient",
		recipientPayload,
		&recipientResponse,
	); err != nil {
		return ports.PayoutInitiationResult{}, fmt.Errorf("create Paystack transfer recipient: %w", err)
	}

	recipientCode := strings.TrimSpace(recipientResponse.RecipientCode)
	if recipientCode == "" {
		return ports.PayoutInitiationResult{}, ErrInvalidResponse
	}

	metadata, err := json.Marshal(map[string]string{
		"payout_id":               payout.ID(),
		"payout_attempt_id":       request.Attempt.ID(),
		"store_id":                payout.StoreID(),
		"order_id":                payout.OrderID(),
		"business_reference_type": payout.BusinessReference().Type(),
		"business_reference_id":   payout.BusinessReference().ID(),
	})
	if err != nil {
		return ports.PayoutInitiationResult{}, fmt.Errorf("encode Paystack payout metadata: %w", err)
	}

	transferPayload := struct {
		Source    string          `json:"source"`
		Amount    int64           `json:"amount"`
		Recipient string          `json:"recipient"`
		Reason    string          `json:"reason"`
		Reference string          `json:"reference"`
		Metadata  json.RawMessage `json:"metadata"`
	}{
		Source: "balance",
		// The domain amount is already denominated in minor units.
		Amount:    payout.Amount().Amount(),
		Recipient: recipientCode,
		Reason:    strings.TrimSpace(payout.Reason()),
		// Each payout attempt gets its own stable provider reference.
		Reference: request.Attempt.ID(),
		Metadata:  metadata,
	}

	var transferResponse struct {
		Reference    string `json:"reference"`
		Status       string `json:"status"`
		TransferCode string `json:"transfer_code"`
	}

	if err := p.client.Request(
		ctx,
		http.MethodPost,
		"/transfer",
		transferPayload,
		&transferResponse,
	); err != nil {
		return ports.PayoutInitiationResult{}, fmt.Errorf("initiate Paystack transfer: %w", err)
	}

	if strings.TrimSpace(transferResponse.Reference) != request.Attempt.ID() {
		return ports.PayoutInitiationResult{}, ErrInvalidResponse
	}

	status := normalizePaystackInitiationStatus(transferResponse.Status)
	providerMetadata, err := json.Marshal(transferResponse)
	if err != nil {
		return ports.PayoutInitiationResult{}, fmt.Errorf("encode Paystack transfer response: %w", err)
	}

	return ports.PayoutInitiationResult{
		ProviderReference: transferResponse.Reference,
		Status:            status,
		Metadata:          providerMetadata,
	}, nil
}

func (p *PayoutProvider) ReconcilePayout(
	ctx context.Context,
	request ports.PayoutReconciliationRequest,
) (ports.PayoutReconciliationResult, error) {
	if p == nil || p.client == nil {
		return ports.PayoutReconciliationResult{}, ErrInvalidConfiguration
	}
	if request.Payout == nil || request.Attempt == nil ||
		request.Attempt.PayoutID() != request.Payout.ID() {
		return ports.PayoutReconciliationResult{}, errors.New("invalid Paystack payout reconciliation request")
	}

	reference := strings.TrimSpace(request.Attempt.ProviderReference())
	if reference == "" {
		// Recover safely when Paystack accepted the transfer but the
		// initiation response was lost.
		reference = request.Attempt.ID()
	}

	var response struct {
		Amount    int64           `json:"amount"`
		Currency  string          `json:"currency"`
		Reference string          `json:"reference"`
		Status    string          `json:"status"`
		Reason    string          `json:"reason"`
		Metadata  json.RawMessage `json:"metadata"`
	}

	path := "/transfer/verify/" + url.PathEscape(reference)
	if err := p.client.Request(ctx, http.MethodGet, path, nil, &response); err != nil {
		return ports.PayoutReconciliationResult{}, fmt.Errorf("verify Paystack transfer: %w", err)
	}

	if strings.TrimSpace(response.Reference) != reference {
		return ports.PayoutReconciliationResult{}, ErrInvalidResponse
	}

	status, failureReason := normalizePaystackTransferStatus(
		response.Status,
		response.Reason,
	)

	metadata := response.Metadata
	if len(metadata) == 0 || !json.Valid(metadata) {
		metadata, _ = json.Marshal(response)
	}

	return ports.PayoutReconciliationResult{
		Amount:            response.Amount,
		Currency:          strings.ToUpper(strings.TrimSpace(response.Currency)),
		ProviderReference: response.Reference,
		Status:            status,
		FailureReason:     failureReason,
		Metadata:          metadata,
	}, nil
}

func normalizePaystackInitiationStatus(status string) enums.PayoutAttemptStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "success":
		return enums.PayoutAttemptStatusSucceeded
	case "pending", "processing", "queued":
		return enums.PayoutAttemptStatusProcessing
	default:
		// OTP and unknown outcomes must be reconciled, never assumed successful.
		return enums.PayoutAttemptStatusRequiresReconciliation
	}
}

func normalizePaystackTransferStatus(
	status string,
	reason string,
) (enums.PayoutAttemptStatus, string) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "success":
		return enums.PayoutAttemptStatusSucceeded, ""
	case "failed", "reversed":
		reason = strings.TrimSpace(reason)
		if reason == "" {
			reason = "Paystack reported that the transfer failed"
		}
		return enums.PayoutAttemptStatusFailed, reason
	case "pending", "processing", "queued", "otp":
		return enums.PayoutAttemptStatusProcessing, ""
	default:
		return enums.PayoutAttemptStatusRequiresReconciliation, ""
	}
}

var _ ports.PayoutProvider = (*PayoutProvider)(nil)
