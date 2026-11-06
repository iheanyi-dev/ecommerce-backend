// Package payment implements Order's outbound Payment service adapter.
package payment

import (
	"context"
	"fmt"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/proto/payment/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/shared/config"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Client calls Payment's authenticated internal gRPC API.
type Client struct {
	client        paymentv1.PaymentServiceClient
	conn          *grpc.ClientConn
	serviceName   string
	serviceSecret string
}

// NewClient creates the Payment gRPC connection and ties its lifetime to Order's Fx app.
func NewClient(lc fx.Lifecycle, cfg *config.Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("create Payment client: config is nil")
	}
	if cfg.PaymentServiceAddr == "" {
		return nil, fmt.Errorf("create Payment client: PAYMENT_SERVICE_ADDR is required")
	}

	conn, err := grpc.NewClient(
		cfg.PaymentServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("create Payment gRPC connection: %w", err)
	}

	client := &Client{
		client:        paymentv1.NewPaymentServiceClient(conn),
		conn:          conn,
		serviceName:   cfg.OrderServiceAuthName,
		serviceSecret: cfg.OrderServiceAuthSecret,
	}

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			if err := conn.Close(); err != nil {
				return fmt.Errorf("close Payment gRPC connection: %w", err)
			}
			return nil
		},
	})

	return client, nil
}

// InitiatePayment forwards an order payment request to the Payment service.
func (c *Client) InitiatePayment(
	ctx context.Context,
	request ports.InitiatePaymentRequest,
) (*ports.InitiatePaymentResponse, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("initiate Payment: client is unavailable")
	}
	if request.PayerID == [16]byte{} || request.BusinessReferenceID == [16]byte{} {
		return nil, fmt.Errorf("initiate Payment: payer and business reference IDs are required")
	}
	if request.AmountMinor <= 0 || request.Currency == "" || request.CustomerEmail == "" {
		return nil, fmt.Errorf("initiate Payment: amount, currency, and customer email are required")
	}

	ctx = metadata.AppendToOutgoingContext(
		ctx,
		"x-service-name", c.serviceName,
		"x-service-secret", c.serviceSecret,
	)

	response, err := c.client.InitiatePayment(ctx, &paymentv1.InitiatePaymentRequest{
		PayerId:               request.PayerID.String(),
		Purpose:               request.Purpose,
		BusinessReferenceType: request.BusinessReferenceType,
		BusinessReferenceId:   request.BusinessReferenceID.String(),
		AmountMinor:           request.AmountMinor,
		Currency:              request.Currency,
		CustomerEmail:         request.CustomerEmail,
	})
	if err != nil {
		return nil, fmt.Errorf("call Payment InitiatePayment RPC: %w", err)
	}
	if response == nil {
		return nil, fmt.Errorf("Payment returned a nil initiation response")
	}

	return &ports.InitiatePaymentResponse{
		PaymentID:   response.GetPaymentId(),
		AttemptID:   response.GetAttemptId(),
		CheckoutURL: response.GetCheckoutUrl(),
	}, nil
}

// InitiatePayout forwards an order settlement request to the Payment service.
func (c *Client) InitiatePayout(
	ctx context.Context,
	request ports.InitiatePayoutRequest,
) (*ports.InitiatePayoutResponse, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("initiate payout: client is unavailable")
	}
	if request.BusinessReferenceID == [16]byte{} ||
		request.StoreID == [16]byte{} ||
		request.OrderID == [16]byte{} {
		return nil, fmt.Errorf("initiate payout: business reference, store, and order IDs are required")
	}
	if strings.TrimSpace(request.BusinessReferenceType) == "" ||
		strings.TrimSpace(request.Reason) == "" ||
		request.AmountMinor <= 0 ||
		strings.TrimSpace(request.Currency) == "" {
		return nil, fmt.Errorf("initiate payout: reference type, reason, positive amount, and currency are required")
	}
	if err := validateBeneficiary(request.Beneficiary); err != nil {
		return nil, fmt.Errorf("initiate payout: %w", err)
	}

	ctx = metadata.AppendToOutgoingContext(
		ctx,
		"x-service-name", c.serviceName,
		"x-service-secret", c.serviceSecret,
	)

	response, err := c.client.InitiatePayout(ctx, &paymentv1.InitiatePayoutRequest{
		BusinessReferenceType: request.BusinessReferenceType,
		BusinessReferenceId:   request.BusinessReferenceID.String(),
		StoreId:               request.StoreID.String(),
		OrderId:               request.OrderID.String(),
		Reason:                request.Reason,
		AmountMinor:           request.AmountMinor,
		Currency:              request.Currency,
		Beneficiary: &paymentv1.PayoutBeneficiary{
			AccountName:   request.Beneficiary.AccountName,
			AccountNumber: request.Beneficiary.AccountNumber,
			BankCode:      request.Beneficiary.BankCode,
			BankName:      request.Beneficiary.BankName,
			CountryCode:   request.Beneficiary.CountryCode,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("call Payment InitiatePayout RPC: %w", err)
	}
	if response == nil {
		return nil, fmt.Errorf("Payment returned a nil payout response")
	}

	return &ports.InitiatePayoutResponse{
		PayoutID:      response.GetPayoutId(),
		AttemptID:     response.GetAttemptId(),
		Reference:     response.GetReference(),
		Status:        response.GetStatus(),
		AttemptStatus: response.GetAttemptStatus(),
		Provider:      response.GetProvider(),
		AmountMinor:   response.GetAmountMinor(),
		Currency:      response.GetCurrency(),
	}, nil
}

func validateBeneficiary(beneficiary ports.PayoutBeneficiary) error {
	fields := map[string]string{
		"account name":   beneficiary.AccountName,
		"account number": beneficiary.AccountNumber,
		"bank code":      beneficiary.BankCode,
		"bank name":      beneficiary.BankName,
		"country code":   beneficiary.CountryCode,
	}
	for name, value := range fields {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("beneficiary %s is required", name)
		}
	}
	return nil
}
