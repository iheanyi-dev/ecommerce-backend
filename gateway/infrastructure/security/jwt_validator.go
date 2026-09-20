package security

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"github.com/iheanyi-dev/ecommerce-backend/gateway/application/ports"
)

// JWTValidator validates access tokens issued by the Identity Service.
//
// The Gateway does not issue, refresh, rotate, or revoke tokens.
//
// Its responsibility is limited to verifying that a client presented a
// valid Identity-issued access token before a protected request is forwarded
// to a downstream business service.
type JWTValidator struct {
	secret []byte
	issuer string
}

// NewJWTValidator creates a Gateway JWT validator.
//
// The Gateway must use the same signing secret and issuer configured in the
// Identity Service because Identity is the authority that issues access
// tokens.
func NewJWTValidator(
	secret string,
	issuer string,
) (*JWTValidator, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, errors.New(
			"JWT validation secret cannot be empty",
		)
	}

	if len(secret) < 32 {
		return nil, errors.New(
			"JWT validation secret must be at least 32 characters",
		)
	}

	if strings.TrimSpace(issuer) == "" {
		return nil, errors.New(
			"JWT issuer cannot be empty",
		)
	}

	return &JWTValidator{
		secret: []byte(secret),
		issuer: issuer,
	}, nil
}

// ValidateAccessToken validates an Identity-issued JWT and extracts the
// trusted authenticated identity.
//
// Validation includes:
//
//   - token signature
//   - HS256 signing algorithm
//   - issuer
//   - expiration and standard registered claims
//   - subject/user ID
//   - role
//
// No raw JWT claims are exposed to downstream application code.
func (v *JWTValidator) ValidateAccessToken(
	ctx context.Context,
	tokenString string,
) (ports.AuthenticatedIdentity, error) {
	if err := ctx.Err(); err != nil {
		return ports.AuthenticatedIdentity{}, err
	}

	tokenString = strings.TrimSpace(tokenString)

	if tokenString == "" {
		return ports.AuthenticatedIdentity{}, errors.New(
			"invalid access token",
		)
	}

	token, err := jwt.Parse(
		tokenString,
		func(token *jwt.Token) (interface{}, error) {
			// Identity issues access tokens using HS256.
			//
			// Never accept another algorithm merely because its signature
			// happens to be valid under the configured secret.
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf(
					"unexpected signing method: %s",
					token.Method.Alg(),
				)
			}

			return v.secret, nil
		},
		jwt.WithIssuer(v.issuer),
	)

	if err != nil || !token.Valid {
		return ports.AuthenticatedIdentity{}, errors.New(
			"invalid access token",
		)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ports.AuthenticatedIdentity{}, errors.New(
			"invalid access token claims",
		)
	}

	subject, ok := claims["sub"].(string)
	if !ok || strings.TrimSpace(subject) == "" {
		return ports.AuthenticatedIdentity{}, errors.New(
			"access token subject is required",
		)
	}

	role, ok := claims["role"].(string)
	if !ok || strings.TrimSpace(role) == "" {
		return ports.AuthenticatedIdentity{}, errors.New(
			"access token role is required",
		)
	}

	return ports.AuthenticatedIdentity{
		UserID: subject,
		Role:   role,
	}, nil
}

// Compile-time assertion.
//
// This guarantees that the infrastructure implementation satisfies the
// Gateway application contract.
var _ ports.AccessTokenValidator = (*JWTValidator)(nil)
