package payment

import (
	"context"
	"fmt"

	paymentv1 "github.com/iheanyi-dev/ecommerce-backend/proto/payment/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/shared/config"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Client implements Billing's outbound Payment application port.
type Client struct {
	client        paymentv1.PaymentServiceClient
	serviceName   string
	serviceSecret string
}

// NewClient creates the authenticated Payment gRPC client and manages its
// connection through the Billing application's Fx lifecycle.
func NewClient(lifecycle fx.Lifecycle, cfg *config.Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("create Payment client: config is nil")
	}
	if cfg.PaymentServiceAddr == "" {
		return nil, fmt.Errorf("create Payment client: PAYMENT_SERVICE_ADDR is required")
	}
	if cfg.ServiceAuthName == "" || cfg.ServiceAuthSecret == "" {
		return nil, fmt.Errorf("create Payment client: service credentials are required")
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
		serviceName:   cfg.ServiceAuthName,
		serviceSecret: cfg.ServiceAuthSecret,
	}

	lifecycle.Append(fx.Hook{
		OnStop: func(context.Context) error {
			if err := conn.Close(); err != nil {
				return fmt.Errorf("close Payment gRPC connection: %w", err)
			}
			return nil
		},
	})

	return client, nil
}

// InitiatePayment requests a subscription checkout from Payment.
func (c *Client) InitiatePayment(
	ctx context.Context,
	request ports.InitiatePaymentRequest,
) (ports.InitiatePaymentResult, error) {
	if c == nil || c.client == nil {
		return ports.InitiatePaymentResult{}, fmt.Errorf("initiate Payment: client is unavailable")
	}

	ctx = metadata.AppendToOutgoingContext(
		ctx,
		"x-service-name", c.serviceName,
		"x-service-secret", c.serviceSecret,
	)

	response, err := c.client.InitiatePayment(ctx, &paymentv1.InitiatePaymentRequest{
		PayerId:               request.PayerID,
		Purpose:               request.Purpose,
		BusinessReferenceType: request.BusinessReferenceType,
		BusinessReferenceId:   request.BusinessReferenceID,
		AmountMinor:           request.AmountMinor,
		Currency:              request.Currency,
		CustomerEmail:         request.CustomerEmail,
	})
	if err != nil {
		return ports.InitiatePaymentResult{}, fmt.Errorf("initiate Payment through gRPC: %w", err)
	}
	if response == nil {
		return ports.InitiatePaymentResult{}, fmt.Errorf("Payment returned a nil initiation response")
	}

	return ports.InitiatePaymentResult{
		PaymentID:     response.GetPaymentId(),
		AttemptID:     response.GetAttemptId(),
		PaymentStatus: response.GetPaymentStatus(),
		AttemptStatus: response.GetAttemptStatus(),
		Provider:      response.GetProvider(),
		CheckoutURL:   response.GetCheckoutUrl(),
		AmountMinor:   response.GetAmountMinor(),
		Currency:      response.GetCurrency(),
	}, nil
}

var _ ports.Payment = (*Client)(nil)
