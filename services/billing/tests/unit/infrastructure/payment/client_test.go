package payment_test

import (
	"context"
	"net"
	"testing"

	paymentv1 "github.com/iheanyi-dev/ecommerce-backend/proto/payment/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	paymentclient "github.com/iheanyi-dev/ecommerce-backend/services/billing/infrastructure/payment"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/shared/config"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type testLifecycle struct {
	hooks []fx.Hook
}

func (l *testLifecycle) Append(hook fx.Hook) {
	l.hooks = append(l.hooks, hook)
}

type paymentServer struct {
	paymentv1.UnimplementedPaymentServiceServer
	t *testing.T
}

func (s *paymentServer) InitiatePayment(
	ctx context.Context,
	req *paymentv1.InitiatePaymentRequest,
) (*paymentv1.InitiatePaymentResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		s.t.Fatal("incoming metadata is missing")
	}
	if got := md.Get("x-service-name"); len(got) != 1 || got[0] != "billing" {
		s.t.Fatalf("service name metadata = %v", got)
	}
	if got := md.Get("x-service-secret"); len(got) != 1 || got[0] != "test-secret" {
		s.t.Fatalf("service secret metadata = %v", got)
	}

	if req.GetPayerId() != "payer-id" ||
		req.GetPurpose() != "subscription" ||
		req.GetBusinessReferenceType() != "subscription" ||
		req.GetBusinessReferenceId() != "subscription-id" ||
		req.GetAmountMinor() != 25000 ||
		req.GetCurrency() != "NGN" ||
		req.GetCustomerEmail() != "customer@example.com" {
		s.t.Fatalf("unexpected Payment request: %+v", req)
	}

	return &paymentv1.InitiatePaymentResponse{
		PaymentId:     "payment-id",
		AttemptId:     "attempt-id",
		PaymentStatus: "pending",
		AttemptStatus: "initialized",
		Provider:      "paystack",
		CheckoutUrl:   "https://checkout.example.com",
		AmountMinor:   25000,
		Currency:      "NGN",
	}, nil
}

func TestInitiatePayment(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	server := grpc.NewServer()
	paymentv1.RegisterPaymentServiceServer(server, &paymentServer{t: t})
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})

	lifecycle := &testLifecycle{}
	client, err := paymentclient.NewClient(lifecycle, &config.Config{
		PaymentServiceAddr: listener.Addr().String(),
		ServiceAuthName:    "billing",
		ServiceAuthSecret:  "test-secret",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if len(lifecycle.hooks) != 1 {
		t.Fatalf("registered lifecycle hooks = %d, want 1", len(lifecycle.hooks))
	}
	t.Cleanup(func() {
		if err := lifecycle.hooks[0].OnStop(context.Background()); err != nil {
			t.Errorf("close Payment client: %v", err)
		}
	})

	result, err := client.InitiatePayment(context.Background(), ports.InitiatePaymentRequest{
		PayerID:               "payer-id",
		Purpose:               "subscription",
		BusinessReferenceType: "subscription",
		BusinessReferenceID:   "subscription-id",
		AmountMinor:           25000,
		Currency:              "NGN",
		CustomerEmail:         "customer@example.com",
	})
	if err != nil {
		t.Fatalf("InitiatePayment() error = %v", err)
	}

	if result.PaymentID != "payment-id" ||
		result.AttemptID != "attempt-id" ||
		result.PaymentStatus != "pending" ||
		result.AttemptStatus != "initialized" ||
		result.Provider != "paystack" ||
		result.CheckoutURL != "https://checkout.example.com" ||
		result.AmountMinor != 25000 ||
		result.Currency != "NGN" {
		t.Fatalf("unexpected result: %+v", result)
	}
}
