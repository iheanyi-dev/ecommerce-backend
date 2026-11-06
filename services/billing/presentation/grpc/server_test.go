package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	billingv1 "github.com/iheanyi-dev/ecommerce-backend/proto/billing/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/value_objects"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestParseID(t *testing.T) {
	valid := uuid.New()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid UUID", valid.String(), false},
		{"empty", "", true},
		{"malformed", "not-a-uuid", true},
		{"nil UUID", uuid.Nil.String(), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseID(tt.input)
			if tt.wantErr {
				if status.Code(err) != codes.InvalidArgument {
					t.Fatalf("expected InvalidArgument, got %v", err)
				}
				return
			}
			if err != nil || got != valid {
				t.Fatalf("parseID() = %v, %v", got, err)
			}
		})
	}
}

func TestMapError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"plan not found", application.ErrPlanNotFound, codes.NotFound},
		{"subscription not found", application.ErrSubscriptionNotFound, codes.NotFound},
		{"plan conflict", application.ErrPlanAlreadyExists, codes.AlreadyExists},
		{"subscription conflict", application.ErrSubscriptionConflict, codes.AlreadyExists},
		{"plan in use", application.ErrPlanInUse, codes.FailedPrecondition},
		{"plan unavailable", application.ErrPlanUnavailable, codes.FailedPrecondition},
		{"payment unavailable", application.ErrPaymentUnavailable, codes.Unavailable},
		{"plan repository unavailable", application.ErrPlanRepositoryUnavailable, codes.Unavailable},
		{"canceled", context.Canceled, codes.Canceled},
		{"deadline", context.DeadlineExceeded, codes.DeadlineExceeded},
		{"unexpected error", errors.New("database connection failed"), codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := status.Code(mapError(tt.err)); got != tt.want {
				t.Fatalf("mapError() code = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlanToProto(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	price, err := value_objects.NewMoney(2500, "NGN")
	if err != nil {
		t.Fatal(err)
	}

	plan := entities.Plan{
		ID:          uuid.New(),
		Name:        "Premium",
		Description: "Premium plan",
		Price:       price,
		Duration:    30 * 24 * time.Hour,
		Capabilities: entities.PlanCapabilities{
			MaxProducts: 5,
			AIEnabled:   true,
		},
		Available: true,
		CreatedAt: now,
		UpdatedAt: now,
		Promotion: &entities.Promotion{
			Name:                "Launch",
			DiscountAmountMinor: 500,
			StartsAt:            now,
			EndsAt:              now.Add(24 * time.Hour),
		},
	}

	got := planToProto(plan)
	if got.Id != plan.ID.String() ||
		got.Price.AmountMinor != 2500 ||
		got.Price.Currency != "NGN" ||
		got.DurationSeconds != int64(plan.Duration/time.Second) ||
		got.Capabilities.MaxProducts != 5 ||
		!got.Capabilities.AiEnabled ||
		got.Promotion == nil ||
		got.Promotion.DiscountAmountMinor != 500 {
		t.Fatalf("unexpected plan mapping: %+v", got)
	}
}

func TestSubscriptionToProto(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	price, err := value_objects.NewMoney(5000, "NGN")
	if err != nil {
		t.Fatal(err)
	}

	subscription := entities.Subscription{
		ID:        uuid.New(),
		StoreID:   uuid.New(),
		PlanID:    uuid.New(),
		Price:     price,
		Duration:  30 * 24 * time.Hour,
		StartedAt: now,
		ExpiresAt: now.Add(30 * 24 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}

	got := subscriptionToProto(subscription)
	if got.Id != subscription.ID.String() ||
		got.StoreId != subscription.StoreID.String() ||
		got.PlanId != subscription.PlanID.String() ||
		got.Price.AmountMinor != 5000 ||
		got.StartedAt == nil ||
		got.ExpiresAt == nil {
		t.Fatalf("unexpected subscription mapping: %+v", got)
	}

	empty := subscriptionToProto(entities.Subscription{})
	if empty.StartedAt != nil || empty.ExpiresAt != nil {
		t.Fatal("zero timestamps should map to nil")
	}
}

func TestPromotionFromProto(t *testing.T) {
	now := timestamppb.New(time.Now().UTC().Truncate(time.Second))
	got := promotionFromProto(&billingv1.Promotion{
		Name:                "Launch",
		DiscountAmountMinor: 500,
		StartsAt:            now,
		EndsAt:              now,
	})

	if got == nil || got.Name != "Launch" ||
		got.DiscountAmountMinor != 500 ||
		!got.StartsAt.Equal(now.AsTime()) ||
		!got.EndsAt.Equal(now.AsTime()) {
		t.Fatalf("unexpected promotion mapping: %+v", got)
	}

	if promotionFromProto(nil) != nil {
		t.Fatal("nil promotion should map to nil")
	}
}

func TestRPCRejectsNilRequests(t *testing.T) {
	server := &Server{}
	ctx := context.Background()

	tests := []struct {
		name string
		call func() error
	}{
		{"create plan", func() error { _, err := server.CreatePlan(ctx, nil); return err }},
		{"update plan", func() error { _, err := server.UpdatePlan(ctx, nil); return err }},
		{"delete plan", func() error { _, err := server.DeletePlan(ctx, nil); return err }},
		{"get plan", func() error { _, err := server.GetPlan(ctx, nil); return err }},
		{"subscribe", func() error { _, err := server.Subscribe(ctx, nil); return err }},
		{"cancel subscription", func() error { _, err := server.CancelSubscription(ctx, nil); return err }},
		{"get subscription", func() error { _, err := server.GetSubscription(ctx, nil); return err }},
		{"list store subscriptions", func() error { _, err := server.ListStoreSubscriptions(ctx, nil); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := status.Code(tt.call()); got != codes.InvalidArgument {
				t.Fatalf("expected InvalidArgument, got %v", got)
			}
		})
	}
}

func TestRPCRejectsInvalidUUIDs(t *testing.T) {
	server := &Server{}
	ctx := context.Background()

	tests := []struct {
		name string
		call func() error
	}{
		{"update plan", func() error {
			_, err := server.UpdatePlan(ctx, &billingv1.UpdatePlanRequest{
				Id: "invalid", Price: &billingv1.Money{AmountMinor: 100, Currency: "NGN"},
				Capabilities: &billingv1.PlanCapabilities{MaxProducts: 1},
			})
			return err
		}},
		{"delete plan", func() error {
			_, err := server.DeletePlan(ctx, &billingv1.DeletePlanRequest{Id: "invalid"})
			return err
		}},
		{"get plan", func() error {
			_, err := server.GetPlan(ctx, &billingv1.GetPlanRequest{Id: "invalid"})
			return err
		}},
		{"subscribe store ID", func() error {
			_, err := server.Subscribe(ctx, &billingv1.SubscribeRequest{
				StoreId: "invalid", PayerId: uuid.NewString(), PlanId: uuid.NewString(),
			})
			return err
		}},
		{"subscribe payer ID", func() error {
			_, err := server.Subscribe(ctx, &billingv1.SubscribeRequest{
				StoreId: uuid.NewString(), PayerId: "invalid", PlanId: uuid.NewString(),
			})
			return err
		}},
		{"subscribe plan ID", func() error {
			_, err := server.Subscribe(ctx, &billingv1.SubscribeRequest{
				StoreId: uuid.NewString(), PayerId: uuid.NewString(), PlanId: "invalid",
			})
			return err
		}},
		{"cancel store ID", func() error {
			_, err := server.CancelSubscription(ctx, &billingv1.CancelSubscriptionRequest{
				StoreId: "invalid", SubscriptionId: uuid.NewString(),
			})
			return err
		}},
		{"cancel subscription ID", func() error {
			_, err := server.CancelSubscription(ctx, &billingv1.CancelSubscriptionRequest{
				StoreId: uuid.NewString(), SubscriptionId: "invalid",
			})
			return err
		}},
		{"get subscription", func() error {
			_, err := server.GetSubscription(ctx, &billingv1.GetSubscriptionRequest{
				SubscriptionId: "invalid",
			})
			return err
		}},
		{"list store subscriptions", func() error {
			_, err := server.ListStoreSubscriptions(ctx, &billingv1.ListStoreSubscriptionsRequest{
				StoreId: "invalid",
			})
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := status.Code(tt.call()); got != codes.InvalidArgument {
				t.Fatalf("expected InvalidArgument, got %v", got)
			}
		})
	}
}

func TestCreatePlanRejectsMissingRequiredFields(t *testing.T) {
	server := &Server{}

	_, err := server.CreatePlan(context.Background(), &billingv1.CreatePlanRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestUpdatePlanRejectsMissingRequiredFields(t *testing.T) {
	server := &Server{}

	_, err := server.UpdatePlan(context.Background(), &billingv1.UpdatePlanRequest{
		Id: uuid.NewString(),
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}
