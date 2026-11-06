package activation_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/activation"
	domain_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

func TestNewToken_CreatesActivationToken(t *testing.T) {
	t.Parallel()

	userID := user.NewUserID()
	tokenHash := strings.Repeat("a", 64)
	expiresAt := time.Now().UTC().Add(time.Hour)

	token, err := activation.NewToken(userID, tokenHash, expiresAt)
	if err != nil {
		t.Fatalf("expected token creation to succeed, got: %v", err)
	}

	if token == nil {
		t.Fatal("expected token to be created")
	}

	if token.ID() == uuid.Nil {
		t.Fatal("expected token ID to be generated")
	}

	if token.UserID() != userID {
		t.Fatal("expected token to belong to supplied user")
	}

	if token.TokenHash() != tokenHash {
		t.Fatal("expected token hash to be stored")
	}

	if !token.ExpiresAt().Equal(expiresAt) {
		t.Fatal("expected token expiry to be preserved")
	}

	if token.CreatedAt().IsZero() {
		t.Fatal("expected creation timestamp to be set")
	}

	if token.IsConsumed() {
		t.Fatal("expected new token to be unconsumed")
	}
}

func TestNewToken_RejectsInvalidUserID(t *testing.T) {
	t.Parallel()

	_, err := activation.NewToken(
		user.UserID{},
		strings.Repeat("a", 64),
		time.Now().Add(time.Hour),
	)

	if err != domain_errors.ErrInvalidUserID {
		t.Fatalf("expected invalid user ID error, got: %v", err)
	}
}

func TestNewToken_RejectsInvalidHash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		hash string
	}{
		{name: "empty hash", hash: ""},
		{name: "short hash", hash: "abc"},
		{name: "invalid hexadecimal", hash: strings.Repeat("z", 64)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := activation.NewToken(
				user.NewUserID(),
				tt.hash,
				time.Now().Add(time.Hour),
			)

			if err != domain_errors.ErrInvalidActivationTokenHash {
				t.Fatalf("expected invalid hash error, got: %v", err)
			}
		})
	}
}

func TestNewToken_RejectsNonFutureExpiry(t *testing.T) {
	t.Parallel()

	_, err := activation.NewToken(
		user.NewUserID(),
		strings.Repeat("a", 64),
		time.Now().Add(-time.Minute),
	)

	if err != domain_errors.ErrInvalidActivationTokenExpiry {
		t.Fatalf("expected invalid expiry error, got: %v", err)
	}
}

func TestToken_Consume(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	token, err := activation.NewToken(
		user.NewUserID(),
		strings.Repeat("a", 64),
		now.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	if err := token.Consume(now); err != nil {
		t.Fatalf("expected token consumption to succeed, got: %v", err)
	}

	if !token.IsConsumed() {
		t.Fatal("expected token to be consumed")
	}

	if consumedAt := token.ConsumedAt(); consumedAt == nil || !consumedAt.Equal(now) {
		t.Fatal("expected consumption timestamp to be recorded")
	}
}

func TestToken_ConsumeRejectsExpiredToken(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	token, err := activation.NewToken(
		user.NewUserID(),
		strings.Repeat("a", 64),
		now.Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	err = token.Consume(now.Add(2 * time.Minute))
	if err != domain_errors.ErrActivationTokenExpired {
		t.Fatalf("expected expired token error, got: %v", err)
	}

	if token.IsConsumed() {
		t.Fatal("expected expired token to remain unconsumed")
	}
}

func TestToken_ConsumeRejectsAlreadyConsumedToken(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	token, err := activation.NewToken(
		user.NewUserID(),
		strings.Repeat("a", 64),
		now.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	if err := token.Consume(now); err != nil {
		t.Fatalf("failed to consume token: %v", err)
	}

	err = token.Consume(now.Add(time.Minute))
	if err != domain_errors.ErrActivationTokenAlreadyConsumed {
		t.Fatalf("expected already consumed error, got: %v", err)
	}
}
