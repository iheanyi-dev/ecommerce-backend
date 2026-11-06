package billing

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc/metadata"

	billingv1 "github.com/iheanyi-dev/ecommerce-backend/proto/billing/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/shared/config"
)

const (
	serviceNameMetadata   = "x-service-name"
	serviceSecretMetadata = "x-service-secret"
)

// Provider implements the Store BillingProvider port using Billing gRPC.
type Provider struct {
	client        billingv1.BillingServiceClient
	serviceName   string
	serviceSecret string
}

// NewProvider constructs a Billing gRPC adapter.
func NewProvider(
	client billingv1.BillingServiceClient,
	serviceName string,
	serviceSecret string,
) *Provider {
	return &Provider{
		client:        client,
		serviceName:   serviceName,
		serviceSecret: serviceSecret,
	}
}

// NewConfiguredProvider builds the Billing adapter using Store's trusted
// internal service credentials.
func NewConfiguredProvider(
	client billingv1.BillingServiceClient,
	cfg *config.Config,
) *Provider {
	return NewProvider(client, cfg.ServiceAuthName, cfg.ServiceAuthSecret)
}

// GetPlan retrieves a Billing plan by ID.
func (p *Provider) GetPlan(
	ctx context.Context,
	planID uuid.UUID,
) (*ports.BillingPlan, error) {
	ctx = p.withServiceMetadata(ctx)

	response, err := p.client.GetPlan(ctx, &billingv1.GetPlanRequest{
		Id: planID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("get plan from Billing: %w", err)
	}
	if response == nil || response.GetPlan() == nil {
		return nil, fmt.Errorf("Billing returned an empty plan response")
	}

	plan, err := mapPlan(response.GetPlan())
	if err != nil {
		return nil, fmt.Errorf("map Billing plan: %w", err)
	}
	return plan, nil
}

// ListPlans retrieves all plans configured in Billing.
func (p *Provider) ListPlans(ctx context.Context) ([]ports.BillingPlan, error) {
	ctx = p.withServiceMetadata(ctx)

	response, err := p.client.ListPlans(ctx, &billingv1.Empty{})
	if err != nil {
		return nil, fmt.Errorf("list Billing plans: %w", err)
	}
	if response == nil {
		return nil, fmt.Errorf("Billing returned an empty plans response")
	}

	plans := make([]ports.BillingPlan, 0, len(response.GetPlans()))
	for _, item := range response.GetPlans() {
		if item == nil {
			return nil, fmt.Errorf("Billing returned a nil plan")
		}
		plan, err := mapPlan(item)
		if err != nil {
			return nil, fmt.Errorf("map Billing plan: %w", err)
		}
		plans = append(plans, *plan)
	}
	return plans, nil
}

// GetSubscription retrieves a subscription by ID.
func (p *Provider) GetSubscription(
	ctx context.Context,
	subscriptionID uuid.UUID,
) (*ports.BillingSubscription, error) {
	ctx = p.withServiceMetadata(ctx)

	response, err := p.client.GetSubscription(ctx, &billingv1.GetSubscriptionRequest{
		SubscriptionId: subscriptionID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("get Billing subscription: %w", err)
	}
	if response == nil || response.GetSubscription() == nil {
		return nil, fmt.Errorf("Billing returned an empty subscription response")
	}

	subscription, err := mapSubscription(response.GetSubscription())
	if err != nil {
		return nil, fmt.Errorf("map Billing subscription: %w", err)
	}
	return subscription, nil
}

// ListStoreSubscriptions retrieves all subscriptions belonging to a Store.
func (p *Provider) ListStoreSubscriptions(
	ctx context.Context,
	storeID uuid.UUID,
) ([]ports.BillingSubscription, error) {
	ctx = p.withServiceMetadata(ctx)

	response, err := p.client.ListStoreSubscriptions(ctx, &billingv1.ListStoreSubscriptionsRequest{
		StoreId: storeID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("list Store Billing subscriptions: %w", err)
	}
	if response == nil {
		return nil, fmt.Errorf("Billing returned an empty subscriptions response")
	}

	subscriptions := make([]ports.BillingSubscription, 0, len(response.GetSubscriptions()))
	for _, item := range response.GetSubscriptions() {
		if item == nil {
			return nil, fmt.Errorf("Billing returned a nil subscription")
		}
		subscription, err := mapSubscription(item)
		if err != nil {
			return nil, fmt.Errorf("map Billing subscription: %w", err)
		}
		subscriptions = append(subscriptions, *subscription)
	}
	return subscriptions, nil
}

// Subscribe asks Billing to create or resume the subscription checkout.
func (p *Provider) Subscribe(
	ctx context.Context,
	input ports.BillingSubscribeInput,
) (*ports.BillingSubscribeOutput, error) {
	ctx = p.withServiceMetadata(ctx)

	response, err := p.client.Subscribe(ctx, &billingv1.SubscribeRequest{
		StoreId:       input.StoreID.String(),
		PayerId:       input.PayerID.String(),
		PlanId:        input.PlanID.String(),
		CustomerEmail: input.CustomerEmail,
	})
	if err != nil {
		return nil, fmt.Errorf("subscribe through Billing: %w", err)
	}
	if response == nil || response.GetSubscription() == nil {
		return nil, fmt.Errorf("Billing returned an empty subscription response")
	}

	subscriptionID, err := uuid.Parse(response.GetSubscription().GetId())
	if err != nil {
		return nil, fmt.Errorf("Billing returned an invalid subscription ID: %w", err)
	}

	paymentID, err := uuid.Parse(response.GetPaymentId())
	if err != nil {
		return nil, fmt.Errorf("Billing returned an invalid payment ID: %w", err)
	}

	return &ports.BillingSubscribeOutput{
		SubscriptionID: subscriptionID,
		PaymentID:      paymentID,
		CheckoutURL:    response.GetCheckoutUrl(),
	}, nil
}

func mapPlan(source *billingv1.Plan) (*ports.BillingPlan, error) {
	if source == nil {
		return nil, fmt.Errorf("plan is required")
	}

	id, err := uuid.Parse(source.GetId())
	if err != nil {
		return nil, fmt.Errorf("invalid plan ID: %w", err)
	}

	result := &ports.BillingPlan{
		ID:          id,
		Type:        source.GetType(),
		Name:        source.GetName(),
		Description: source.GetDescription(),
		Duration:    time.Duration(source.GetDurationSeconds()) * time.Second,
		Available:   source.GetAvailable(),
		CreatedAt:   timestampValue(source.GetCreatedAt()),
		UpdatedAt:   timestampValue(source.GetUpdatedAt()),
	}

	if price := source.GetPrice(); price != nil {
		result.Price = ports.BillingMoney{
			AmountMinor: price.GetAmountMinor(),
			Currency:    price.GetCurrency(),
		}
	}

	if capabilities := source.GetCapabilities(); capabilities != nil {
		result.Capabilities = ports.BillingPlanCapabilities{
			MaxProducts: int(capabilities.GetMaxProducts()),
			AIEnabled:   capabilities.GetAiEnabled(),
		}
	}

	if promotion := source.GetPromotion(); promotion != nil {
		result.Promotion = &ports.BillingPlanPromotion{
			Name:                promotion.GetName(),
			DiscountAmountMinor: promotion.GetDiscountAmountMinor(),
			StartsAt:            timestampValue(promotion.GetStartsAt()),
			EndsAt:              timestampValue(promotion.GetEndsAt()),
		}
	}

	return result, nil
}

func mapSubscription(source *billingv1.Subscription) (*ports.BillingSubscription, error) {
	if source == nil {
		return nil, fmt.Errorf("subscription is required")
	}

	id, err := uuid.Parse(source.GetId())
	if err != nil {
		return nil, fmt.Errorf("invalid subscription ID: %w", err)
	}

	storeID, err := uuid.Parse(source.GetStoreId())
	if err != nil {
		return nil, fmt.Errorf("invalid subscription Store ID: %w", err)
	}

	planID, err := uuid.Parse(source.GetPlanId())
	if err != nil {
		return nil, fmt.Errorf("invalid subscription plan ID: %w", err)
	}

	result := &ports.BillingSubscription{
		ID:        id,
		StoreID:   storeID,
		PlanID:    planID,
		Duration:  time.Duration(source.GetDurationSeconds()) * time.Second,
		Status:    source.GetStatus(),
		StartedAt: timestampValue(source.GetStartedAt()),
		ExpiresAt: timestampValue(source.GetExpiresAt()),
		CreatedAt: timestampValue(source.GetCreatedAt()),
		UpdatedAt: timestampValue(source.GetUpdatedAt()),
	}

	if price := source.GetPrice(); price != nil {
		result.Price = ports.BillingMoney{
			AmountMinor: price.GetAmountMinor(),
			Currency:    price.GetCurrency(),
		}
	}

	return result, nil
}

func timestampValue(value interface{ AsTime() time.Time }) *time.Time {
	if value == nil {
		return nil
	}
	result := value.AsTime()
	return &result
}

func (p *Provider) withServiceMetadata(ctx context.Context) context.Context {
	requestMetadata := metadata.MD{}

	if existing, ok := metadata.FromOutgoingContext(ctx); ok {
		requestMetadata = existing.Copy()
	}

	requestMetadata.Set(serviceNameMetadata, p.serviceName)
	requestMetadata.Set(serviceSecretMetadata, p.serviceSecret)

	otel.GetTextMapPropagator().Inject(
		ctx,
		metadataCarrier{metadata: requestMetadata},
	)

	return metadata.NewOutgoingContext(ctx, requestMetadata)
}

type metadataCarrier struct {
	metadata metadata.MD
}

func (c metadataCarrier) Get(key string) string {
	values := c.metadata.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (c metadataCarrier) Set(key, value string) {
	c.metadata.Set(key, value)
}

func (c metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(c.metadata))
	for key := range c.metadata {
		keys = append(keys, key)
	}
	return keys
}

var _ propagation.TextMapCarrier = metadataCarrier{}
var _ ports.BillingProvider = (*Provider)(nil)
