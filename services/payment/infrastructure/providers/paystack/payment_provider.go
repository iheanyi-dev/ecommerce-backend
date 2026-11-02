package paystack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
)

type PaymentProvider struct {
	client *Client
}

func NewPaymentProvider(client *Client) (*PaymentProvider, error) {
	if client == nil {
		return nil, ErrInvalidConfiguration
	}
	return &PaymentProvider{client: client}, nil
}

func (p *PaymentProvider) InitiateCheckout(
	ctx context.Context,
	request ports.CheckoutRequest,
) (ports.CheckoutResult, error) {
	if p == nil || p.client == nil {
		return ports.CheckoutResult{}, ErrInvalidConfiguration
	}

	email := strings.TrimSpace(request.CustomerEmail)
	parsedEmail, err := mail.ParseAddress(email)
	if err != nil || parsedEmail.Address != email ||
		strings.TrimSpace(request.PaymentID) == "" ||
		strings.TrimSpace(request.AttemptID) == "" ||
		request.Amount.IsZero() {
		return ports.CheckoutResult{}, errors.New("invalid Paystack checkout request")
	}

	metadata, err := json.Marshal(map[string]string{
		"payment_id":         request.PaymentID,
		"payment_attempt_id": request.AttemptID,
	})
	if err != nil {
		return ports.CheckoutResult{}, fmt.Errorf("encode Paystack metadata: %w", err)
	}

	payload := struct {
		Email     string          `json:"email"`
		Amount    int64           `json:"amount"`
		Currency  string          `json:"currency"`
		Reference string          `json:"reference"`
		Metadata  json.RawMessage `json:"metadata"`
	}{
		Email: email,
		// Money already uses minor units. Never multiply this amount by 100.
		Amount:   request.Amount.Amount(),
		Currency: request.Amount.Currency(),
		// The attempt UUID is the stable, idempotent provider reference.
		Reference: request.AttemptID,
		Metadata:  metadata,
	}

	var response struct {
		AuthorizationURL string `json:"authorization_url"`
		AccessCode       string `json:"access_code"`
		Reference        string `json:"reference"`
	}

	if err := p.client.Request(
		ctx,
		http.MethodPost,
		"/transaction/initialize",
		payload,
		&response,
	); err != nil {
		return ports.CheckoutResult{}, fmt.Errorf("initialize Paystack transaction: %w", err)
	}

	if strings.TrimSpace(response.AuthorizationURL) == "" ||
		strings.TrimSpace(response.Reference) == "" ||
		response.Reference != request.AttemptID {
		return ports.CheckoutResult{}, ErrInvalidResponse
	}

	providerMetadata, err := json.Marshal(response)
	if err != nil {
		return ports.CheckoutResult{}, fmt.Errorf("encode Paystack checkout response: %w", err)
	}

	return ports.CheckoutResult{
		ProviderReference: response.Reference,
		CheckoutURL:       response.AuthorizationURL,
		Status:            enums.PaymentAttemptStatusPending,
		Metadata:          providerMetadata,
	}, nil
}

func (p *PaymentProvider) ReconcileAttempt(
	ctx context.Context,
	request ports.ReconciliationRequest,
) (ports.ReconciliationResult, error) {
	if p == nil || p.client == nil {
		return ports.ReconciliationResult{}, ErrInvalidConfiguration
	}

	if request.Payment == nil || request.Attempt == nil ||
		request.Payment.ID() == "" ||
		request.Attempt.ID() == "" ||
		request.Attempt.PaymentID() != request.Payment.ID() {
		return ports.ReconciliationResult{}, errors.New("invalid payment reconciliation request")
	}

	reference := strings.TrimSpace(request.Attempt.ProviderReference())
	if reference == "" {
		// The initiation response may have been lost. The attempt UUID
		// is also the reference originally sent to Paystack.
		reference = request.Attempt.ID()
	}

	var response struct {
		Status          string          `json:"status"`
		Reference       string          `json:"reference"`
		Amount          int64           `json:"amount"`
		Currency        string          `json:"currency"`
		GatewayResponse string          `json:"gateway_response"`
		Message         string          `json:"message"`
		Metadata        json.RawMessage `json:"metadata"`
	}

	path := "/transaction/verify/" + url.PathEscape(reference)
	if err := p.client.Request(ctx, http.MethodGet, path, nil, &response); err != nil {
		return ports.ReconciliationResult{}, fmt.Errorf("verify Paystack transaction: %w", err)
	}

	status, failureReason := normalizePaystackStatus(
		response.Status,
		response.GatewayResponse,
		response.Message,
	)

	metadata := response.Metadata
	if len(metadata) == 0 || !json.Valid(metadata) {
		metadata, _ = json.Marshal(response)
	}

	return ports.ReconciliationResult{
		Amount:            response.Amount,
		Currency:          strings.ToUpper(strings.TrimSpace(response.Currency)),
		ProviderReference: response.Reference,
		Status:            status,
		FailureReason:     failureReason,
		Metadata:          metadata,
	}, nil
}

func normalizePaystackStatus(
	status string,
	gatewayResponse string,
	message string,
) (enums.PaymentAttemptStatus, string) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "success":
		return enums.PaymentAttemptStatusSucceeded, ""

	case "failed", "abandoned":
		reason := strings.TrimSpace(gatewayResponse)
		if reason == "" {
			reason = strings.TrimSpace(message)
		}
		if reason == "" {
			reason = "Paystack reported that the transaction failed"
		}
		return enums.PaymentAttemptStatusFailed, reason

	case "pending", "ongoing", "processing", "queued":
		return enums.PaymentAttemptStatusProcessing, ""

	default:
		// Unknown provider statuses must never be treated as successful.
		return enums.PaymentAttemptStatusRequiresReconciliation, ""
	}
}
