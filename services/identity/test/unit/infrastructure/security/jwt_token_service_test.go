package security_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/security"
)

const (
	testJWTSecret = "test-secret"
	testJWTIssuer = "identity-service"
)

// newTestJWTService creates a JWT service using deterministic test
// configuration.
//
// Keeping this setup in one helper prevents individual tests from
// accidentally using different JWT configuration.
func newTestJWTService(t *testing.T) *security.JWTTokenService {
	t.Helper()

	service, err := security.NewJWTTokenService(
		security.TokenConfig{
			Secret:         testJWTSecret,
			Issuer:         testJWTIssuer,
			AccessTokenTTL: 15 * time.Minute,
		},
	)
	if err != nil {
		t.Fatalf("failed to create token service: %v", err)
	}

	return service
}

func TestJWTTokenService_GenerateAccessToken(t *testing.T) {
	service := newTestJWTService(t)

	userID := user.NewUserID()
	role := user.RoleVendor

	tokenString, err := service.GenerateAccessToken(
		context.Background(),
		userID,
		role,
	)
	if err != nil {
		t.Fatalf(
			"expected token generation to succeed, got %v",
			err,
		)
	}

	if tokenString == "" {
		t.Fatal("expected generated token not to be empty")
	}

	token, err := jwt.Parse(
		tokenString,
		func(token *jwt.Token) (interface{}, error) {
			if token.Method != jwt.SigningMethodHS256 {
				t.Fatalf(
					"expected HS256 signing method, got %v",
					token.Method,
				)
			}

			return []byte(testJWTSecret), nil
		},
	)
	if err != nil {
		t.Fatalf("failed to parse generated token: %v", err)
	}

	if !token.Valid {
		t.Fatal("expected generated token to be valid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("expected JWT MapClaims")
	}

	if claims["sub"] != userID.String() {
		t.Fatalf(
			"expected subject %q, got %v",
			userID.String(),
			claims["sub"],
		)
	}

	if claims["role"] != role.String() {
		t.Fatalf(
			"expected role %q, got %v",
			role.String(),
			claims["role"],
		)
	}

	if claims["iss"] != testJWTIssuer {
		t.Fatalf(
			"expected issuer %q, got %v",
			testJWTIssuer,
			claims["iss"],
		)
	}
}

// TestJWTTokenService_ValidateAccessToken verifies that a token generated
// by the service can be validated and converted into an application-level
// authenticated identity.
func TestJWTTokenService_ValidateAccessToken(t *testing.T) {
	service := newTestJWTService(t)

	userID := user.NewUserID()
	role := user.RoleVendor

	tokenString, err := service.GenerateAccessToken(
		context.Background(),
		userID,
		role,
	)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	identity, err := service.ValidateAccessToken(
		context.Background(),
		tokenString,
	)
	if err != nil {
		t.Fatalf(
			"expected token validation to succeed, got %v",
			err,
		)
	}

	if identity.UserID != userID.String() {
		t.Fatalf(
			"expected user ID %q, got %q",
			userID.String(),
			identity.UserID,
		)
	}

	if identity.Role != role.String() {
		t.Fatalf(
			"expected role %q, got %q",
			role.String(),
			identity.Role,
		)
	}
}

// TestJWTTokenService_ValidateAccessToken_RejectsEmptyToken verifies that
// an empty token can never be treated as authenticated.
func TestJWTTokenService_ValidateAccessToken_RejectsEmptyToken(t *testing.T) {
	service := newTestJWTService(t)

	_, err := service.ValidateAccessToken(
		context.Background(),
		"",
	)

	if err == nil {
		t.Fatal("expected empty token to be rejected")
	}
}

// TestJWTTokenService_ValidateAccessToken_RejectsInvalidToken verifies
// that malformed JWT strings are rejected.
func TestJWTTokenService_ValidateAccessToken_RejectsInvalidToken(t *testing.T) {
	service := newTestJWTService(t)

	_, err := service.ValidateAccessToken(
		context.Background(),
		"this-is-not-a-jwt",
	)

	if err == nil {
		t.Fatal("expected invalid token to be rejected")
	}
}

// TestJWTTokenService_ValidateAccessToken_RejectsWrongSecret verifies
// cryptographic signature validation.
func TestJWTTokenService_ValidateAccessToken_RejectsWrongSecret(t *testing.T) {
	service := newTestJWTService(t)

	userID := user.NewUserID()

	otherService, err := security.NewJWTTokenService(
		security.TokenConfig{
			Secret:         "different-secret",
			Issuer:         testJWTIssuer,
			AccessTokenTTL: 15 * time.Minute,
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to create second token service: %v",
			err,
		)
	}

	tokenString, err := otherService.GenerateAccessToken(
		context.Background(),
		userID,
		user.RoleUser,
	)
	if err != nil {
		t.Fatalf(
			"failed to generate token with different secret: %v",
			err,
		)
	}

	_, err = service.ValidateAccessToken(
		context.Background(),
		tokenString,
	)

	if err == nil {
		t.Fatal("expected token signed with wrong secret to be rejected")
	}
}

// TestJWTTokenService_ValidateAccessToken_RejectsExpiredToken verifies
// that expired access tokens cannot authenticate a request.
func TestJWTTokenService_ValidateAccessToken_RejectsExpiredToken(t *testing.T) {
	service := newTestJWTService(t)

	now := time.Now()

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			"sub":  user.NewUserID().String(),
			"role": user.RoleUser.String(),
			"iss":  testJWTIssuer,
			"iat":  now.Add(-2 * time.Hour).Unix(),
			"exp":  now.Add(-1 * time.Hour).Unix(),
		},
	)

	tokenString, err := token.SignedString(
		[]byte(testJWTSecret),
	)
	if err != nil {
		t.Fatalf(
			"failed to sign expired token: %v",
			err,
		)
	}

	_, err = service.ValidateAccessToken(
		context.Background(),
		tokenString,
	)

	if err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}

// TestJWTTokenService_ValidateAccessToken_RejectsWrongIssuer verifies
// that a token issued by another issuer cannot authenticate against the
// Identity service.
func TestJWTTokenService_ValidateAccessToken_RejectsWrongIssuer(t *testing.T) {
	service := newTestJWTService(t)

	now := time.Now()

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			"sub":  user.NewUserID().String(),
			"role": user.RoleUser.String(),
			"iss":  "different-service",
			"iat":  now.Unix(),
			"exp":  now.Add(15 * time.Minute).Unix(),
		},
	)

	tokenString, err := token.SignedString(
		[]byte(testJWTSecret),
	)
	if err != nil {
		t.Fatalf(
			"failed to sign token: %v",
			err,
		)
	}

	_, err = service.ValidateAccessToken(
		context.Background(),
		tokenString,
	)

	if err == nil {
		t.Fatal("expected token with wrong issuer to be rejected")
	}
}

// TestJWTTokenService_ValidateAccessToken_RejectsMissingSubject verifies
// that the authenticated user identity must always be present.
func TestJWTTokenService_ValidateAccessToken_RejectsMissingSubject(t *testing.T) {
	service := newTestJWTService(t)

	now := time.Now()

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			"role": user.RoleUser.String(),
			"iss":  testJWTIssuer,
			"iat":  now.Unix(),
			"exp":  now.Add(15 * time.Minute).Unix(),
		},
	)

	tokenString, err := token.SignedString(
		[]byte(testJWTSecret),
	)
	if err != nil {
		t.Fatalf(
			"failed to sign token: %v",
			err,
		)
	}

	_, err = service.ValidateAccessToken(
		context.Background(),
		tokenString,
	)

	if err == nil {
		t.Fatal("expected token without subject to be rejected")
	}
}

// TestJWTTokenService_ValidateAccessToken_RejectsMissingRole verifies
// that authorization information must be present in the authenticated
// identity.
func TestJWTTokenService_ValidateAccessToken_RejectsMissingRole(t *testing.T) {
	service := newTestJWTService(t)

	now := time.Now()

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			"sub": user.NewUserID().String(),
			"iss": testJWTIssuer,
			"iat": now.Unix(),
			"exp": now.Add(15 * time.Minute).Unix(),
		},
	)

	tokenString, err := token.SignedString(
		[]byte(testJWTSecret),
	)
	if err != nil {
		t.Fatalf(
			"failed to sign token: %v",
			err,
		)
	}

	_, err = service.ValidateAccessToken(
		context.Background(),
		tokenString,
	)

	if err == nil {
		t.Fatal("expected token without role to be rejected")
	}
}

// TestJWTTokenService_ValidateAccessToken_RejectsUnexpectedSigningMethod
// verifies that the service does not accept a token using an algorithm
// other than HS256.
func TestJWTTokenService_ValidateAccessToken_RejectsUnexpectedSigningMethod(
	t *testing.T,
) {
	service := newTestJWTService(t)

	now := time.Now()

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS384,
		jwt.MapClaims{
			"sub":  user.NewUserID().String(),
			"role": user.RoleUser.String(),
			"iss":  testJWTIssuer,
			"iat":  now.Unix(),
			"exp":  now.Add(15 * time.Minute).Unix(),
		},
	)

	tokenString, err := token.SignedString(
		[]byte(testJWTSecret),
	)
	if err != nil {
		t.Fatalf(
			"failed to sign token using HS384: %v",
			err,
		)
	}

	_, err = service.ValidateAccessToken(
		context.Background(),
		tokenString,
	)

	if err == nil {
		t.Fatal(
			"expected token using unexpected signing method to be rejected",
		)
	}
}

// TestJWTTokenService_ValidateAccessToken_RespectsCancelledContext
// verifies that validation does not continue when the request context
// has already been cancelled.
func TestJWTTokenService_ValidateAccessToken_RespectsCancelledContext(
	t *testing.T,
) {
	service := newTestJWTService(t)

	userID := user.NewUserID()

	tokenString, err := service.GenerateAccessToken(
		context.Background(),
		userID,
		user.RoleUser,
	)
	if err != nil {
		t.Fatalf(
			"failed to generate access token: %v",
			err,
		)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = service.ValidateAccessToken(
		ctx,
		tokenString,
	)

	if err == nil {
		t.Fatal("expected cancelled context to return an error")
	}

	if err != context.Canceled {
		t.Fatalf(
			"expected context.Canceled, got %v",
			err,
		)
	}
}

func TestNewJWTTokenService_RejectsEmptySecret(t *testing.T) {
	_, err := security.NewJWTTokenService(
		security.TokenConfig{
			Secret:         "",
			Issuer:         testJWTIssuer,
			AccessTokenTTL: 15 * time.Minute,
		},
	)

	if err == nil {
		t.Fatal("expected empty JWT secret to be rejected")
	}
}

func TestNewJWTTokenService_RejectsEmptyIssuer(t *testing.T) {
	_, err := security.NewJWTTokenService(
		security.TokenConfig{
			Secret:         testJWTSecret,
			Issuer:         "",
			AccessTokenTTL: 15 * time.Minute,
		},
	)

	if err == nil {
		t.Fatal("expected empty JWT issuer to be rejected")
	}
}

func TestNewJWTTokenService_RejectsInvalidTTL(t *testing.T) {
	_, err := security.NewJWTTokenService(
		security.TokenConfig{
			Secret:         testJWTSecret,
			Issuer:         testJWTIssuer,
			AccessTokenTTL: 0,
		},
	)

	if err == nil {
		t.Fatal("expected invalid JWT TTL to be rejected")
	}
}

func TestJWTTokenService_RespectsCancelledContext(t *testing.T) {
	service := newTestJWTService(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := service.GenerateAccessToken(
		ctx,
		user.NewUserID(),
		user.RoleUser,
	)

	if err == nil {
		t.Fatal("expected cancelled context to return an error")
	}

	if err != context.Canceled {
		t.Fatalf(
			"expected context.Canceled, got %v",
			err,
		)
	}
}