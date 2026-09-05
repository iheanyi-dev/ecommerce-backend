package use_cases

import (
	"context"
	"strings"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
)

// LogoutUserUseCase revokes the refresh-token session represented
// by the refresh token supplied by the client.
//
// Each refresh token represents an independent device/session.
// Therefore, logout revokes only this specific token and does not
// affect other active sessions belonging to the same user.
type LogoutUserUseCase struct {
	refreshTokenRepository ports.RefreshTokenRepository
	refreshTokenService    ports.RefreshTokenService
}

// NewLogoutUserUseCase creates a logout use case with its required
// refresh-token dependencies.
func NewLogoutUserUseCase(
	refreshTokenRepository ports.RefreshTokenRepository,
	refreshTokenService ports.RefreshTokenService,
) *LogoutUserUseCase {
	return &LogoutUserUseCase{
		refreshTokenRepository: refreshTokenRepository,
		refreshTokenService:    refreshTokenService,
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
		return ErrInvalidRefreshToken
	}

	// Refresh tokens are stored as hashes, never as raw tokens.
	tokenHash, err := u.refreshTokenService.Hash(ctx, refreshToken)
	if err != nil {
		return ErrRefreshTokenHashing
	}

	// Locate the exact refresh-token session represented by this token.
	record, err := u.refreshTokenRepository.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return ErrInvalidRefreshToken
	}

	if record == nil {
		return ErrInvalidRefreshToken
	}

	// An already-revoked token cannot be used to perform another logout.
	if record.RevokedAt != nil {
		return ErrInvalidRefreshToken
	}

	// Revoke only this refresh-token session.
	if err := u.refreshTokenRepository.Revoke(
		ctx,
		record.ID,
		time.Now(),
	); err != nil {
		return ErrRefreshTokenRevocation
	}

	return nil
}

// Compile-time assertion that LogoutUserUseCase satisfies the
// application service contract exposed to the presentation layer.
var _ ports.LogoutUserService = (*LogoutUserUseCase)(nil)
