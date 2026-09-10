package use_cases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
)

type fakeLogoutLogger struct {
	events []ports.LogEvent
}

func (f *fakeLogoutLogger) Log(
	ctx context.Context,
	event ports.LogEvent,
) error {
	f.events = append(f.events, event)
	return nil
}

func (f *fakeLogoutLogger) lastEvent(t *testing.T) ports.LogEvent {
	t.Helper()

	if len(f.events) == 0 {
		t.Fatal("expected at least one logout log event, got none")
	}

	return f.events[len(f.events)-1]
}

func assertLogoutEvent(
	t *testing.T,
	logger *fakeLogoutLogger,
	expectedEvent string,
	expectedOperation string,
	expectedUserID string,
	expectedFailureCategory string,
) {
	t.Helper()

	if len(logger.events) != 1 {
		t.Fatalf(
			"expected exactly one logout log event, got %d",
			len(logger.events),
		)
	}

	event := logger.lastEvent(t)

	if event.Event != expectedEvent {
		t.Fatalf(
			"expected event %q, got %q",
			expectedEvent,
			event.Event,
		)
	}

	if event.Operation != expectedOperation {
		t.Fatalf(
			"expected operation %q, got %q",
			expectedOperation,
			event.Operation,
		)
	}

	if event.UserID != expectedUserID {
		t.Fatalf(
			"expected user ID %q, got %q",
			expectedUserID,
			event.UserID,
		)
	}

	if event.FailureCategory != expectedFailureCategory {
		t.Fatalf(
			"expected failure category %q, got %q",
			expectedFailureCategory,
			event.FailureCategory,
		)
	}
}

func assertLogoutEventContainsNoSecrets(
	t *testing.T,
	event ports.LogEvent,
	secrets ...string,
) {
	t.Helper()

	values := []string{
		event.Event,
		event.Operation,
		event.UserID,
		event.Role,
		event.HTTPMethod,
		event.Route,
		event.FailureCategory,
		event.RequiredRole,
	}

	for _, secret := range secrets {
		for _, value := range values {
			if value == secret {
				t.Fatalf(
					"expected logout log event not to contain sensitive value %q",
					secret,
				)
			}
		}
	}
}

var _ ports.Logger = (*fakeLogoutLogger)(nil)

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

	logger := &fakeLogoutLogger{}

	useCase := use_cases.NewLogoutUserUseCase(
		repository,
		refreshTokenService,
		logger,
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

	assertLogoutEvent(
		t,
		logger,
		"auth.logout.succeeded",
		"logout",
		"user-1",
		"",
	)

	assertLogoutEventContainsNoSecrets(
		t,
		logger.lastEvent(t),
		refreshToken,
		tokenHash,
	)
}

func TestLogoutUserUseCase_Logout_RejectsBlankRefreshToken(t *testing.T) {
	t.Parallel()

	refreshToken := "   "

	refreshTokenService := &mockRefreshTokenService{}
	repository := &mockRefreshTokenRepository{}
	logger := &fakeLogoutLogger{}

	useCase := use_cases.NewLogoutUserUseCase(
		repository,
		refreshTokenService,
		logger,
	)

	err := useCase.Logout(
		context.Background(),
		refreshToken,
	)

	if !errors.Is(err, application_errors.ErrInvalidRefreshToken) {
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

	assertLogoutEvent(
		t,
		logger,
		"auth.logout.invalid_token",
		"logout",
		"",
		"invalid_refresh_token",
	)

	assertLogoutEventContainsNoSecrets(
		t,
		logger.lastEvent(t),
		refreshToken,
	)
}

func TestLogoutUserUseCase_Logout_RejectsUnknownRefreshToken(t *testing.T) {
	t.Parallel()

	refreshToken := "unknown-refresh-token"

	refreshTokenService := &mockRefreshTokenService{
		hashedToken: "hashed-unknown-token",
	}

	repository := &mockRefreshTokenRepository{
		record: nil,
	}

	logger := &fakeLogoutLogger{}

	useCase := use_cases.NewLogoutUserUseCase(
		repository,
		refreshTokenService,
		logger,
	)

	err := useCase.Logout(
		context.Background(),
		refreshToken,
	)

	if !errors.Is(err, application_errors.ErrInvalidRefreshToken) {
		t.Fatalf(
			"expected ErrInvalidRefreshToken, got %v",
			err,
		)
	}

	if repository.revokedID != "" {
		t.Fatal("expected no refresh token to be revoked")
	}

	assertLogoutEvent(
		t,
		logger,
		"auth.logout.invalid_token",
		"logout",
		"",
		"invalid_refresh_token",
	)

	assertLogoutEventContainsNoSecrets(
		t,
		logger.lastEvent(t),
		refreshToken,
		"hashed-unknown-token",
	)
}

func TestLogoutUserUseCase_Logout_RejectsAlreadyRevokedRefreshToken(t *testing.T) {
	t.Parallel()

	now := time.Now()
	revokedAt := now.Add(-time.Minute)
	refreshToken := "already-revoked-token"
	tokenHash := "hashed-revoked-token"

	refreshTokenService := &mockRefreshTokenService{
		hashedToken: tokenHash,
	}

	repository := &mockRefreshTokenRepository{
		record: &ports.RefreshTokenRecord{
			ID:        "session-1",
			UserID:    "user-1",
			TokenHash: tokenHash,
			ExpiresAt: now.Add(24 * time.Hour),
			RevokedAt: &revokedAt,
			CreatedAt: now.Add(-time.Hour),
		},
	}

	logger := &fakeLogoutLogger{}

	useCase := use_cases.NewLogoutUserUseCase(
		repository,
		refreshTokenService,
		logger,
	)

	err := useCase.Logout(
		context.Background(),
		refreshToken,
	)

	if !errors.Is(err, application_errors.ErrInvalidRefreshToken) {
		t.Fatalf(
			"expected ErrInvalidRefreshToken, got %v",
			err,
		)
	}

	if repository.revokedID != "" {
		t.Fatal("expected already-revoked refresh token not to be revoked again")
	}

	assertLogoutEvent(
		t,
		logger,
		"auth.logout.invalid_token",
		"logout",
		"user-1",
		"invalid_refresh_token",
	)

	assertLogoutEventContainsNoSecrets(
		t,
		logger.lastEvent(t),
		refreshToken,
		tokenHash,
	)
}

func TestLogoutUserUseCase_Logout_RevokesExpiredRefreshToken(t *testing.T) {
	t.Parallel()

	now := time.Now()
	tokenID := "expired-session-1"
	refreshToken := "expired-refresh-token"
	tokenHash := "hashed-expired-token"

	refreshTokenService := &mockRefreshTokenService{
		hashedToken: tokenHash,
	}

	repository := &mockRefreshTokenRepository{
		record: &ports.RefreshTokenRecord{
			ID:        tokenID,
			UserID:    "user-1",
			TokenHash: tokenHash,
			ExpiresAt: now.Add(-time.Hour),
			CreatedAt: now.Add(-24 * time.Hour),
		},
	}

	logger := &fakeLogoutLogger{}

	useCase := use_cases.NewLogoutUserUseCase(
		repository,
		refreshTokenService,
		logger,
	)

	err := useCase.Logout(
		context.Background(),
		refreshToken,
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

	assertLogoutEvent(
		t,
		logger,
		"auth.logout.succeeded",
		"logout",
		"user-1",
		"",
	)

	assertLogoutEventContainsNoSecrets(
		t,
		logger.lastEvent(t),
		refreshToken,
		tokenHash,
	)
}

func TestLogoutUserUseCase_Logout_ReturnsHashingError(t *testing.T) {
	t.Parallel()

	refreshToken := "valid-refresh-token"

	refreshTokenService := &mockRefreshTokenService{
		hashErr: errors.New("hash failed"),
	}

	repository := &mockRefreshTokenRepository{}
	logger := &fakeLogoutLogger{}

	useCase := use_cases.NewLogoutUserUseCase(
		repository,
		refreshTokenService,
		logger,
	)

	err := useCase.Logout(
		context.Background(),
		refreshToken,
	)

	if !errors.Is(err, application_errors.ErrRefreshTokenHashing) {
		t.Fatalf(
			"expected ErrRefreshTokenHashing, got %v",
			err,
		)
	}

	if repository.revokedID != "" {
		t.Fatal("expected no refresh token to be revoked")
	}

	assertLogoutEvent(
		t,
		logger,
		"auth.logout.failed",
		"logout",
		"",
		"token_hashing_failed",
	)

	assertLogoutEventContainsNoSecrets(
		t,
		logger.lastEvent(t),
		refreshToken,
	)
}

func TestLogoutUserUseCase_Logout_ReturnsRevocationError(t *testing.T) {
	t.Parallel()

	now := time.Now()
	refreshToken := "valid-refresh-token"
	tokenHash := "hashed-refresh-token"

	refreshTokenService := &mockRefreshTokenService{
		hashedToken: tokenHash,
	}

	repository := &mockRefreshTokenRepository{
		record: &ports.RefreshTokenRecord{
			ID:        "session-1",
			UserID:    "user-1",
			TokenHash: tokenHash,
			ExpiresAt: now.Add(24 * time.Hour),
			CreatedAt: now.Add(-time.Hour),
		},
		revokeErr: errors.New("database revoke failed"),
	}

	logger := &fakeLogoutLogger{}

	useCase := use_cases.NewLogoutUserUseCase(
		repository,
		refreshTokenService,
		logger,
	)

	err := useCase.Logout(
		context.Background(),
		refreshToken,
	)

	if !errors.Is(err, application_errors.ErrRefreshTokenRevocation) {
		t.Fatalf(
			"expected ErrRefreshTokenRevocation, got %v",
			err,
		)
	}

	assertLogoutEvent(
		t,
		logger,
		"auth.logout.failed",
		"logout",
		"user-1",
		"token_revocation_failed",
	)

	assertLogoutEventContainsNoSecrets(
		t,
		logger.lastEvent(t),
		refreshToken,
		tokenHash,
	)
}

func TestLogoutUserUseCase_Logout_RevokesOnlySpecifiedSession(t *testing.T) {
	t.Parallel()

	now := time.Now()
	refreshToken := "device-2-refresh-token"
	tokenHash := "hashed-device-2-token"

	refreshTokenService := &mockRefreshTokenService{
		hashedToken: tokenHash,
	}

	repository := &mockRefreshTokenRepository{
		record: &ports.RefreshTokenRecord{
			ID:        "device-2-session",
			UserID:    "user-1",
			TokenHash: tokenHash,
			ExpiresAt: now.Add(24 * time.Hour),
			CreatedAt: now.Add(-time.Hour),
		},
	}

	logger := &fakeLogoutLogger{}

	useCase := use_cases.NewLogoutUserUseCase(
		repository,
		refreshTokenService,
		logger,
	)

	err := useCase.Logout(
		context.Background(),
		refreshToken,
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

	if len(logger.events) != 1 {
		t.Fatalf(
			"expected exactly one logout log event, got %d",
			len(logger.events),
		)
	}
}
