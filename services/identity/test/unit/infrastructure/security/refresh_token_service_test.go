package security_test

import (
	"context"
	"strings"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/security"
)

func TestRefreshTokenService_Generate(t *testing.T) {
	t.Parallel()

	service := security.NewRefreshTokenService()

	token, err := service.Generate(context.Background())
	if err != nil {
		t.Fatalf("expected token generation to succeed, got error: %v", err)
	}

	if strings.TrimSpace(token) == "" {
		t.Fatal("expected generated refresh token not to be empty")
	}

	if len(token) < 40 {
		t.Fatalf(
			"expected generated refresh token to have sufficient length, got %d",
			len(token),
		)
	}
}

func TestRefreshTokenService_GenerateProducesUniqueTokens(t *testing.T) {
	t.Parallel()

	service := security.NewRefreshTokenService()

	firstToken, err := service.Generate(context.Background())
	if err != nil {
		t.Fatalf("failed to generate first token: %v", err)
	}

	secondToken, err := service.Generate(context.Background())
	if err != nil {
		t.Fatalf("failed to generate second token: %v", err)
	}

	if firstToken == secondToken {
		t.Fatal("expected independently generated refresh tokens to differ")
	}
}

func TestRefreshTokenService_Hash(t *testing.T) {
	t.Parallel()

	service := security.NewRefreshTokenService()

	token := "refresh-token"

	firstHash, err := service.Hash(
		context.Background(),
		token,
	)
	if err != nil {
		t.Fatalf("expected hashing to succeed, got error: %v", err)
	}

	secondHash, err := service.Hash(
		context.Background(),
		token,
	)
	if err != nil {
		t.Fatalf("expected hashing to succeed, got error: %v", err)
	}

	if firstHash == "" {
		t.Fatal("expected hash not to be empty")
	}

	if firstHash != secondHash {
		t.Fatal("expected hashing the same token to produce the same hash")
	}

	if firstHash == token {
		t.Fatal("expected persisted representation not to equal raw token")
	}
}

func TestRefreshTokenService_HashProducesDifferentHashesForDifferentTokens(
	t *testing.T,
) {
	t.Parallel()

	service := security.NewRefreshTokenService()

	firstHash, err := service.Hash(
		context.Background(),
		"first-refresh-token",
	)
	if err != nil {
		t.Fatalf("failed to hash first token: %v", err)
	}

	secondHash, err := service.Hash(
		context.Background(),
		"second-refresh-token",
	)
	if err != nil {
		t.Fatalf("failed to hash second token: %v", err)
	}

	if firstHash == secondHash {
		t.Fatal("expected different tokens to produce different hashes")
	}
}

func TestRefreshTokenService_HashRejectsEmptyToken(t *testing.T) {
	t.Parallel()

	service := security.NewRefreshTokenService()

	_, err := service.Hash(
		context.Background(),
		"",
	)

	if err == nil {
		t.Fatal("expected empty refresh token hashing to fail")
	}
}

