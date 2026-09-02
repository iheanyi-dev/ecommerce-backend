package security

import "time"

// TokenConfig contains the configuration required to issue access tokens.
//
// Keeping token configuration in Infrastructure prevents the Application
// layer from knowing anything about JWT implementation details.
type TokenConfig struct {
	// Secret is the private signing key used to sign access tokens.
	Secret string

	// Issuer identifies the service that issued the token.
	Issuer string

	// AccessTokenTTL determines how long an access token remains valid.
	AccessTokenTTL time.Duration
}
