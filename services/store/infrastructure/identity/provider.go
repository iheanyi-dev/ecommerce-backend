package identity

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc/metadata"

	identityv1 "github.com/iheanyi-dev/ecommerce-backend/proto/identity/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/shared/config"
)

const (
	serviceNameMetadata   = "x-service-name"
	serviceSecretMetadata = "x-service-secret"
)

// Provider implements the Store application IdentityProvider port using the
// Identity Service gRPC API.
type Provider struct {
	client        identityv1.IdentityServiceClient
	serviceName   string
	serviceSecret string
}

// NewProvider creates an Identity Service gRPC-backed IdentityProvider.
func NewProvider(
	client identityv1.IdentityServiceClient,
	serviceName string,
	serviceSecret string,
) *Provider {
	return &Provider{
		client:        client,
		serviceName:   serviceName,
		serviceSecret: serviceSecret,
	}
}

// NewConfiguredProvider builds the Identity provider using Store's existing
// trusted service-auth credentials.
func NewConfiguredProvider(
	client identityv1.IdentityServiceClient,
	cfg *config.Config,
) *Provider {
	return NewProvider(
		client,
		cfg.ServiceAuthName,
		cfg.ServiceAuthSecret,
	)
}

// IsVendor determines whether the supplied user currently has vendor status.
func (p *Provider) IsVendor(
	ctx context.Context,
	userID uuid.UUID,
) (bool, error) {
	ctx = p.withServiceMetadata(ctx)

	response, err := p.client.GetUser(
		ctx,
		&identityv1.GetUserRequest{
			UserId: userID.String(),
		},
	)
	if err != nil {
		return false, err
	}

	if response == nil || response.GetUser() == nil {
		return false, fmt.Errorf(
			"identity service returned an empty user response",
		)
	}

	return response.GetUser().GetRole() == "vendor", nil
}

// PromoteToVendor promotes the supplied user to vendor status.
func (p *Provider) PromoteToVendor(
	ctx context.Context,
	userID uuid.UUID,
) error {
	ctx = p.withServiceMetadata(ctx)

	_, err := p.client.PromoteToVendor(
		ctx,
		&identityv1.PromoteToVendorRequest{
			UserId: userID.String(),
		},
	)

	return err
}

func (p *Provider) withServiceMetadata(ctx context.Context) context.Context {
	requestMetadata := metadata.MD{}

	if existing, ok := metadata.FromOutgoingContext(ctx); ok {
		requestMetadata = existing.Copy()
	}

	requestMetadata.Set(
		serviceNameMetadata,
		p.serviceName,
	)
	requestMetadata.Set(
		serviceSecretMetadata,
		p.serviceSecret,
	)

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

func (c metadataCarrier) Set(key string, value string) {
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

var _ ports.IdentityProvider = (*Provider)(nil)
