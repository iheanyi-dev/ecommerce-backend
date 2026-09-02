package security

import (
	"fmt"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/shared/config"
)

// NewJWTTokenServiceFromConfig creates the JWT TokenService using the
// application's centralized configuration.
//
// Configuration is translated into the Infrastructure-specific
// TokenConfig here, keeping JWT implementation details outside the
// shared configuration package.
func NewJWTTokenServiceFromConfig(
	cfg *config.Config,
) (*JWTTokenService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}

	return NewJWTTokenService(
		TokenConfig{
			Secret:         cfg.JWTSecret,
			Issuer:         cfg.JWTIssuer,
			AccessTokenTTL: cfg.JWTAccessTokenTTL,
		},
	)
}