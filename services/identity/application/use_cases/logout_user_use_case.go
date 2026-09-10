package use_cases

import (
	"context"
	"strings"
	"time"

	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
)

// LogoutUserUseCase revokes the refresh-token session represented
// by the refresh token supplied by the client.
//
// Each refresh token represents an independent device/session.
// Therefore, logout revokes only this specific token and does not
// affect other active sessions belonging to the same user.
//
// The use case also records structured authentication events through
// the application logger. Raw refresh tokens and token hashes are
// deliberately excluded from all log events.
type LogoutUserUseCase struct {
	refreshTokenRepository ports.RefreshTokenRepository
	refreshTokenService    ports.RefreshTokenService
	logger                 ports.Logger
}

// NewLogoutUserUseCase creates a logout use case with its required
// refresh-token and observability dependencies.
func NewLogoutUserUseCase(
	refreshTokenRepository ports.RefreshTokenRepository,
	refreshTokenService ports.RefreshTokenService,
	logger ports.Logger,
) *LogoutUserUseCase {
	return &LogoutUserUseCase{
		refreshTokenRepository: refreshTokenRepository,
		refreshTokenService:    refreshTokenService,
		logger:                 logger,
	}
}

// Logout revokes the refresh-token session represented by the
// supplied raw refresh token.
func (u *LogoutUserUseCase) Logout(
	ctx context.Context,
	refreshToken string,
) error {
	// A blank token cannot identify a valid session.
	if strings.TrimSpace(refreshToken) == "" {
		u.logLogoutEvent(ctx, ports.LogEvent{
			Event:           "auth.logout.invalid_token",
			Operation:       "logout",
			FailureCategory: "invalid_refresh_token",
		})

		return application_errors.ErrInvalidRefreshToken
	}

	// Refresh tokens are stored as hashes, never as raw tokens.
	tokenHash, err := u.refreshTokenService.Hash(ctx, refreshToken)
	if err != nil {
		u.logLogoutEvent(ctx, ports.LogEvent{
			Event:           "auth.logout.failed",
			Operation:       "logout",
			FailureCategory: "token_hashing_failed",
		})

		return application_errors.ErrRefreshTokenHashing
	}

	// Locate the exact refresh-token session represented by this token.
	record, err := u.refreshTokenRepository.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		u.logLogoutEvent(ctx, ports.LogEvent{
			Event:           "auth.logout.invalid_token",
			Operation:       "logout",
			FailureCategory: "invalid_refresh_token",
		})

		return application_errors.ErrInvalidRefreshToken
	}

	if record == nil {
		u.logLogoutEvent(ctx, ports.LogEvent{
			Event:           "auth.logout.invalid_token",
			Operation:       "logout",
			FailureCategory: "invalid_refresh_token",
		})

		return application_errors.ErrInvalidRefreshToken
	}

	// An already-revoked token cannot be used to perform another logout.
	if record.RevokedAt != nil {
		u.logLogoutEvent(ctx, ports.LogEvent{
			Event:           "auth.logout.invalid_token",
			Operation:       "logout",
			UserID:          record.UserID,
			FailureCategory: "invalid_refresh_token",
		})

		return application_errors.ErrInvalidRefreshToken
	}

	// Revoke only this refresh-token session.
	if err := u.refreshTokenRepository.Revoke(
		ctx,
		record.ID,
		time.Now(),
	); err != nil {
		u.logLogoutEvent(ctx, ports.LogEvent{
			Event:           "auth.logout.failed",
			Operation:       "logout",
			UserID:          record.UserID,
			FailureCategory: "token_revocation_failed",
		})

		return application_errors.ErrRefreshTokenRevocation
	}

	// The logout operation completed successfully.
	//
	// The event contains only the safe user identity. The raw refresh
	// token and its hash are never included in the log event.
	u.logLogoutEvent(ctx, ports.LogEvent{
		Event:     "auth.logout.succeeded",
		Operation: "logout",
		UserID:    record.UserID,
	})

	return nil
}

// logLogoutEvent records a structured logout event.
//
// Logging is intentionally best-effort. A failure in the observability
// infrastructure must not change the result of the logout operation.
func (u *LogoutUserUseCase) logLogoutEvent(
	ctx context.Context,
	event ports.LogEvent,
) {
	if u.logger == nil {
		return
	}

	_ = u.logger.Log(ctx, event)
}

// Compile-time assertion that LogoutUserUseCase satisfies the
// application service contract exposed to the presentation layer.
var _ ports.LogoutUserService = (*LogoutUserUseCase)(nil)
