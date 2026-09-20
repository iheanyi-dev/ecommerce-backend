package security_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/iheanyi-dev/ecommerce-backend/gateway/infrastructure/security"
)

const testJWTSecret = "this-is-a-valid-development-jwt-secret-123456"

func TestJWTValidator_ValidateAccessToken(t *testing.T) {
	t.Run("valid token", func(t *testing.T) {
		validator, err := security.NewJWTValidator(
			testJWTSecret,
			"identity-service",
		)
		require.NoError(t, err)

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":  "user-123",
			"role": "user",
			"iss":  "identity-service",
			"iat":  time.Now().Unix(),
			"exp":  time.Now().Add(15 * time.Minute).Unix(),
		})

		signedToken, err := token.SignedString([]byte(testJWTSecret))
		require.NoError(t, err)

		identity, err := validator.ValidateAccessToken(
			context.Background(),
			signedToken,
		)

		require.NoError(t, err)
		require.Equal(t, "user-123", identity.UserID)
		require.Equal(t, "user", identity.Role)
	})

	t.Run("wrong signature", func(t *testing.T) {
		validator, err := security.NewJWTValidator(
			testJWTSecret,
			"identity-service",
		)
		require.NoError(t, err)

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":  "user-123",
			"role": "user",
			"iss":  "identity-service",
			"exp":  time.Now().Add(15 * time.Minute).Unix(),
		})

		signedToken, err := token.SignedString(
			[]byte("different-secret-123456789012345678"),
		)
		require.NoError(t, err)

		_, err = validator.ValidateAccessToken(
			context.Background(),
			signedToken,
		)

		require.Error(t, err)
	})

	t.Run("wrong issuer", func(t *testing.T) {
		validator, err := security.NewJWTValidator(
			testJWTSecret,
			"identity-service",
		)
		require.NoError(t, err)

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":  "user-123",
			"role": "user",
			"iss":  "different-service",
			"exp":  time.Now().Add(15 * time.Minute).Unix(),
		})

		signedToken, err := token.SignedString([]byte(testJWTSecret))
		require.NoError(t, err)

		_, err = validator.ValidateAccessToken(
			context.Background(),
			signedToken,
		)

		require.Error(t, err)
	})

	t.Run("wrong signing algorithm", func(t *testing.T) {
		validator, err := security.NewJWTValidator(
			testJWTSecret,
			"identity-service",
		)
		require.NoError(t, err)

		token := jwt.NewWithClaims(jwt.SigningMethodHS384, jwt.MapClaims{
			"sub":  "user-123",
			"role": "user",
			"iss":  "identity-service",
			"exp":  time.Now().Add(15 * time.Minute).Unix(),
		})

		signedToken, err := token.SignedString([]byte(testJWTSecret))
		require.NoError(t, err)

		_, err = validator.ValidateAccessToken(
			context.Background(),
			signedToken,
		)

		require.Error(t, err)
	})

	t.Run("expired token", func(t *testing.T) {
		validator, err := security.NewJWTValidator(
			testJWTSecret,
			"identity-service",
		)
		require.NoError(t, err)

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":  "user-123",
			"role": "user",
			"iss":  "identity-service",
			"exp":  time.Now().Add(-15 * time.Minute).Unix(),
		})

		signedToken, err := token.SignedString([]byte(testJWTSecret))
		require.NoError(t, err)

		_, err = validator.ValidateAccessToken(
			context.Background(),
			signedToken,
		)

		require.Error(t, err)
	})

	t.Run("missing user id", func(t *testing.T) {
		validator, err := security.NewJWTValidator(
			testJWTSecret,
			"identity-service",
		)
		require.NoError(t, err)

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"role": "user",
			"iss":  "identity-service",
			"exp":  time.Now().Add(15 * time.Minute).Unix(),
		})

		signedToken, err := token.SignedString([]byte(testJWTSecret))
		require.NoError(t, err)

		_, err = validator.ValidateAccessToken(
			context.Background(),
			signedToken,
		)

		require.Error(t, err)
	})

	t.Run("missing role", func(t *testing.T) {
		validator, err := security.NewJWTValidator(
			testJWTSecret,
			"identity-service",
		)
		require.NoError(t, err)

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "user-123",
			"iss": "identity-service",
			"exp": time.Now().Add(15 * time.Minute).Unix(),
		})

		signedToken, err := token.SignedString([]byte(testJWTSecret))
		require.NoError(t, err)

		_, err = validator.ValidateAccessToken(
			context.Background(),
			signedToken,
		)

		require.Error(t, err)
	})
}
