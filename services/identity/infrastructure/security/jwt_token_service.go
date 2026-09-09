package security

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// JWTTokenService implements the application's TokenService contract
// using JSON Web Tokens.
//
// JWT is deliberately implemented in Infrastructure. The Application
// layer only knows about ports.TokenService.
type JWTTokenService struct {
	config TokenConfig
}

const minJWTSecretLength = 32

// NewJWTTokenService creates a JWT token service.
//
// The signing secret must never be empty because issuing unsigned or
// improperly signed authentication tokens would compromise the system.
func NewJWTTokenService(config TokenConfig) (*JWTTokenService, error) {
	if strings.TrimSpace(config.Secret) == "" {
		return nil, errors.New("JWT signing secret cannot be empty")
	}

	if strings.TrimSpace(config.Issuer) == "" {
		return nil, errors.New("JWT issuer cannot be empty")
	}
	if len(config.Secret) < minJWTSecretLength {
		return nil, errors.New("JWT signing secret must be at least 32 characters")
	}

	if config.AccessTokenTTL <= 0 {
		return nil, errors.New("JWT access token TTL must be greater than zero")
	}

	return &JWTTokenService{
		config: config,
	}, nil
}

// GenerateAccessToken creates a signed JWT for an authenticated user.
//
// The token contains only the claims required by the authentication and
// authorization system. Sensitive information such as passwords is never
// placed inside the token.
func (s *JWTTokenService) GenerateAccessToken(
	ctx context.Context,
	userID user.UserID,
	role user.Role,
) (string, error) {
	// Respect request cancellation before performing token generation.
	if err := ctx.Err(); err != nil {
		return "", err
	}

	now := time.Now()

	claims := jwt.MapClaims{
		"sub":  userID.String(),
		"role": role.String(),
		"iss":  s.config.Issuer,
		"iat":  now.Unix(),
		"exp":  now.Add(s.config.AccessTokenTTL).Unix(),
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	signedToken, err := token.SignedString(
		[]byte(s.config.Secret),
	)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}

	return signedToken, nil
}

// ValidateAccessToken validates a signed JWT and extracts the authenticated
// user's identity.
//
// JWT-specific concerns remain entirely inside Infrastructure:
//
//   - signature verification
//   - signing algorithm verification
//   - issuer validation
//   - expiration validation
//   - claim extraction
//
// The Presentation layer receives only the application-level identity.
func (s *JWTTokenService) ValidateAccessToken(
	ctx context.Context,
	tokenString string,
) (ports.AuthenticatedIdentity, error) {
	// Respect request cancellation before performing token validation.
	if err := ctx.Err(); err != nil {
		return ports.AuthenticatedIdentity{}, err
	}

	if strings.TrimSpace(tokenString) == "" {
		// Treat an empty access token as an invalid authentication credential.
		return ports.AuthenticatedIdentity{}, application_errors.ErrInvalidAccessToken
	}

	token, err := jwt.Parse(
		tokenString,
		func(token *jwt.Token) (interface{}, error) {
			// Never accept a token using an unexpected signing algorithm.
			//
			// Our access tokens are issued using HMAC SHA-256, so accepting
			// another algorithm could create a serious authentication flaw.
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf(
					"unexpected signing method: %s",
					token.Method.Alg(),
				)
			}

			return []byte(s.config.Secret), nil
		},
		jwt.WithIssuer(s.config.Issuer),
	)
	if err != nil {
		// Expose a stable application-level error to callers while preserving
		// the underlying JWT error for infrastructure-level diagnostics.
		return ports.AuthenticatedIdentity{}, fmt.Errorf(
			"%w: %v",
			application_errors.ErrInvalidAccessToken,
			err,
		)
	}

	if !token.Valid {
		// Return the centralized application error so the presentation layer
		// can consistently translate invalid tokens into HTTP 401 responses.
		return ports.AuthenticatedIdentity{}, application_errors.ErrInvalidAccessToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ports.AuthenticatedIdentity{}, errors.New(
			"invalid access token claims",
		)
	}

	// The subject identifies the authenticated user.
	subject, ok := claims["sub"].(string)
	if !ok || strings.TrimSpace(subject) == "" {
		// A token without a valid subject cannot establish an authenticated
		// user identity.
		return ports.AuthenticatedIdentity{}, application_errors.ErrInvalidAccessToken
	}

	// The role is required by the authorization layer.
	role, ok := claims["role"].(string)
	if !ok || strings.TrimSpace(role) == "" {
		// A token without a valid role cannot safely participate in
		// authorization decisions.
		return ports.AuthenticatedIdentity{}, application_errors.ErrInvalidAccessToken
	}

	return ports.AuthenticatedIdentity{
		UserID: subject,
		Role:   role,
	}, nil
}

// Compile-time assertion.
//
// This guarantees that the infrastructure implementation satisfies the
// application-level TokenService contract.
var _ ports.TokenService = (*JWTTokenService)(nil)
