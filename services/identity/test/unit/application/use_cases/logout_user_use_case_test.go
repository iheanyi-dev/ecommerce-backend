package use_cases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
)

func TestLogoutUserUseCase_Logout_Success(t *testing.T) {
	t.Parallel()

	now := time.Now()
	refreshToken := "valid-refresh-token"
	tokenHash := "hashed-refresh-token"
	tokenID := "session-1"

	refreshTokenService := &mockRefreshTokenService{
		hashedToken: tokenHash,
	}

	repository := &mockRefreshTokenRepository{
		record: &ports.RefreshTokenRecord{
			ID:        tokenID,
			UserID:    "user-1",
			TokenHash: tokenHash,
			ExpiresAt: now.Add(24 * time.Hour),
			CreatedAt: now.Add(-time.Hour),
		},
	}

	useCase := use_cases.NewLogoutUserUseCase(
		repository,
		refreshTokenService,
	)

	err := useCase.Logout(
		context.Background(),
		refreshToken,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repository.revokedID != tokenID {
		t.Fatalf(
			"expected refresh token %q to be revoked, got %q",
			tokenID,
			repository.revokedID,
		)
	}
}

func TestLogoutUserUseCase_Logout_RejectsBlankRefreshToken(t *testing.T) {
	t.Parallel()

	refreshTokenService := &mockRefreshTokenService{}
	repository := &mockRefreshTokenRepository{}

	useCase := use_cases.NewLogoutUserUseCase(
		repository,
		refreshTokenService,
	)

	err := useCase.Logout(
		context.Background(),
		"   ",
	)

	if !errors.Is(err, use_cases.ErrInvalidRefreshToken) {
		t.Fatalf(
			"expected ErrInvalidRefreshToken, got %v",
			err,
		)
	}

	if refreshTokenService.hashedToken != "" {
		t.Fatal("expected refresh token not to be hashed")
	}

	if repository.revokedID != "" {
		t.Fatal("expected no refresh token to be revoked")
	}
}

func TestLogoutUserUseCase_Logout_RejectsUnknownRefreshToken(t *testing.T) {
	t.Parallel()

	refreshTokenService := &mockRefreshTokenService{
		hashedToken: "hashed-unknown-token",
	}

	repository := &mockRefreshTokenRepository{
		record: nil,
	}

	useCase := use_cases.NewLogoutUserUseCase(
		repository,
		refreshTokenService,
	)

	err := useCase.Logout(
		context.Background(),
		"unknown-refresh-token",
	)

	if !errors.Is(err, use_cases.ErrInvalidRefreshToken) {
		t.Fatalf(
			"expected ErrInvalidRefreshToken, got %v",
			err,
		)
	}

	if repository.revokedID != "" {
		t.Fatal("expected no refresh token to be revoked")
	}
}

func TestLogoutUserUseCase_Logout_RejectsAlreadyRevokedRefreshToken(t *testing.T) {
	t.Parallel()

	now := time.Now()
	revokedAt := now.Add(-time.Minute)

	refreshTokenService := &mockRefreshTokenService{
		hashedToken: "hashed-revoked-token",
	}

	repository := &mockRefreshTokenRepository{
		record: &ports.RefreshTokenRecord{
			ID:        "session-1",
			UserID:    "user-1",
			TokenHash: "hashed-revoked-token",
			ExpiresAt: now.Add(24 * time.Hour),
			RevokedAt: &revokedAt,
			CreatedAt: now.Add(-time.Hour),
		},
	}

	useCase := use_cases.NewLogoutUserUseCase(
		repository,
		refreshTokenService,
	)

	err := useCase.Logout(
		context.Background(),
		"already-revoked-token",
	)

	if !errors.Is(err, use_cases.ErrInvalidRefreshToken) {
		t.Fatalf(
			"expected ErrInvalidRefreshToken, got %v",
			err,
		)
	}

	if repository.revokedID != "" {
		t.Fatal("expected already-revoked refresh token not to be revoked again")
	}
}

func TestLogoutUserUseCase_Logout_RevokesExpiredRefreshToken(t *testing.T) {
	t.Parallel()

	now := time.Now()
	tokenID := "expired-session-1"

	refreshTokenService := &mockRefreshTokenService{
		hashedToken: "hashed-expired-token",
	}

	repository := &mockRefreshTokenRepository{
		record: &ports.RefreshTokenRecord{
			ID:        tokenID,
			UserID:    "user-1",
			TokenHash: "hashed-expired-token",
			ExpiresAt: now.Add(-time.Hour),
			CreatedAt: now.Add(-24 * time.Hour),
		},
	}

	useCase := use_cases.NewLogoutUserUseCase(
		repository,
		refreshTokenService,
	)

	err := useCase.Logout(
		context.Background(),
		"expired-refresh-token",
	)

	if err != nil {
		t.Fatalf(
			"expected expired refresh token to be revoked without error, got %v",
			err,
		)
	}

	if repository.revokedID != tokenID {
		t.Fatalf(
			"expected expired refresh token %q to be revoked, got %q",
			tokenID,
			repository.revokedID,
		)
	}
}

func TestLogoutUserUseCase_Logout_ReturnsHashingError(t *testing.T) {
	t.Parallel()

	hashErr := errors.New("hash failed")

	refreshTokenService := &mockRefreshTokenService{
		hashErr: hashErr,
	}

	repository := &mockRefreshTokenRepository{}

	useCase := use_cases.NewLogoutUserUseCase(
		repository,
		refreshTokenService,
	)

	err := useCase.Logout(
		context.Background(),
		"valid-refresh-token",
	)

	if !errors.Is(err, use_cases.ErrRefreshTokenHashing) {
		t.Fatalf(
			"expected ErrRefreshTokenHashing, got %v",
			err,
		)
	}

	if repository.revokedID != "" {
		t.Fatal("expected no refresh token to be revoked")
	}
}

func TestLogoutUserUseCase_Logout_ReturnsRevocationError(t *testing.T) {
	t.Parallel()

	now := time.Now()

	refreshTokenService := &mockRefreshTokenService{
		hashedToken: "hashed-refresh-token",
	}

	repository := &mockRefreshTokenRepository{
		record: &ports.RefreshTokenRecord{
			ID:        "session-1",
			UserID:    "user-1",
			TokenHash: "hashed-refresh-token",
			ExpiresAt: now.Add(24 * time.Hour),
			CreatedAt: now.Add(-time.Hour),
		},
		revokeErr: errors.New("database revoke failed"),
	}

	useCase := use_cases.NewLogoutUserUseCase(
		repository,
		refreshTokenService,
	)

	err := useCase.Logout(
		context.Background(),
		"valid-refresh-token",
	)

	if !errors.Is(err, use_cases.ErrRefreshTokenRevocation) {
		t.Fatalf(
			"expected ErrRefreshTokenRevocation, got %v",
			err,
		)
	}
}

func TestLogoutUserUseCase_Logout_RevokesOnlySpecifiedSession(t *testing.T) {
	t.Parallel()

	now := time.Now()

	refreshTokenService := &mockRefreshTokenService{
		hashedToken: "hashed-device-2-token",
	}

	repository := &mockRefreshTokenRepository{
		record: &ports.RefreshTokenRecord{
			ID:        "device-2-session",
			UserID:    "user-1",
			TokenHash: "hashed-device-2-token",
			ExpiresAt: now.Add(24 * time.Hour),
			CreatedAt: now.Add(-time.Hour),
		},
	}

	useCase := use_cases.NewLogoutUserUseCase(
		repository,
		refreshTokenService,
	)

	err := useCase.Logout(
		context.Background(),
		"device-2-refresh-token",
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repository.revokedID != "device-2-session" {
		t.Fatalf(
			"expected only device-2 session %q to be revoked, got %q",
			"device-2-session",
			repository.revokedID,
		)
	}
}
