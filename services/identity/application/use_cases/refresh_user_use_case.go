package use_cases

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// RefreshUserUseCase coordinates the refresh-token authentication workflow.
//
// The workflow is:
//
//	Raw refresh token
//	    ↓
//	Hash token
//	    ↓
//	Find persisted token session
//	    ↓
//	Validate revocation and expiration
//	    ↓
//	Load current user
//	    ↓
//	Validate current account status
//	    ↓
//	Generate new refresh token
//	    ↓
//	Hash new refresh token
//	    ↓
//	Generate new access token
//	    ↓
//	Revoke old refresh token and persist replacement
//	    ↓
//	Return both tokens
//
// The use case deliberately depends only on application ports. It has no
// knowledge of PostgreSQL, JWT, bcrypt, crypto/rand, or HTTP.
type RefreshUserUseCase struct {
	refreshTokenRepository ports.RefreshTokenRepository
	userRepository         ports.UserRepository
	refreshTokenService    ports.RefreshTokenService
	tokenService           ports.TokenService
}

// NewRefreshUserUseCase creates the refresh-token use case.
//
// Dependencies are supplied through application ports so that the use case
// remains independent from infrastructure implementations.
func NewRefreshUserUseCase(
	refreshTokenRepository ports.RefreshTokenRepository,
	userRepository ports.UserRepository,
	refreshTokenService ports.RefreshTokenService,
	tokenService ports.TokenService,
) *RefreshUserUseCase {
	return &RefreshUserUseCase{
		refreshTokenRepository: refreshTokenRepository,
		userRepository:         userRepository,
		refreshTokenService:    refreshTokenService,
		tokenService:           tokenService,
	}
}

// Refresh validates an existing refresh token and rotates it into a new
// refresh-token session and access token.
func (u *RefreshUserUseCase) Refresh(
	ctx context.Context,
	command dto.RefreshTokenCommand,
) (dto.RefreshTokenResult, error) {
	if strings.TrimSpace(command.RefreshToken) == "" {
		return dto.RefreshTokenResult{}, application_errors.ErrInvalidRefreshToken
	}

	// -------------------------------------------------------------------------
	// 1. Hash the supplied refresh token.
	//
	// Only the hash is used to locate the persisted refresh-token session.
	// The raw refresh token must never be persisted.
	// -------------------------------------------------------------------------

	tokenHash, err := u.refreshTokenService.Hash(
		ctx,
		command.RefreshToken,
	)
	if err != nil {
		return dto.RefreshTokenResult{}, application_errors.ErrRefreshTokenHashing
	}

	// -------------------------------------------------------------------------
	// 2. Locate the refresh-token session.
	// -------------------------------------------------------------------------

	record, err := u.refreshTokenRepository.FindByTokenHash(
		ctx,
		tokenHash,
	)
	if err != nil {
		return dto.RefreshTokenResult{}, application_errors.ErrInvalidRefreshToken
	}

	if record == nil {
		return dto.RefreshTokenResult{}, application_errors.ErrInvalidRefreshToken
	}

	// -------------------------------------------------------------------------
	// 3. Reject revoked or expired refresh tokens.
	// -------------------------------------------------------------------------

	if record.RevokedAt != nil {
		return dto.RefreshTokenResult{}, application_errors.ErrInvalidRefreshToken
	}

	if !record.ExpiresAt.After(time.Now()) {
		return dto.RefreshTokenResult{}, application_errors.ErrInvalidRefreshToken
	}

	// -------------------------------------------------------------------------
	// 4. Reconstruct the user's ID from the persisted session.
	// -------------------------------------------------------------------------

	userID, err := user.UserIDFromString(record.UserID)
	if err != nil {
		return dto.RefreshTokenResult{}, application_errors.ErrInvalidRefreshToken
	}

	// -------------------------------------------------------------------------
	// 5. Load the current user.
	//
	// We deliberately obtain the current user rather than trusting information
	// from the refresh-token record. This means role and account status are
	// evaluated from the current source of truth.
	// -------------------------------------------------------------------------

	currentUser, err := u.userRepository.FindByID(
		ctx,
		userID,
	)
	if err != nil {
		return dto.RefreshTokenResult{}, application_errors.ErrInvalidRefreshToken
	}

	if currentUser == nil {
		return dto.RefreshTokenResult{}, application_errors.ErrInvalidRefreshToken
	}

	if currentUser.Status() != user.StatusActive {
		return dto.RefreshTokenResult{}, application_errors.ErrInvalidRefreshToken
	}

	// -------------------------------------------------------------------------
	// 6. Generate the replacement refresh token BEFORE revoking the old one.
	//
	// If generation fails, the existing refresh session remains usable.
	// -------------------------------------------------------------------------

	newRefreshToken, err := u.refreshTokenService.Generate(ctx)
	if err != nil {
		return dto.RefreshTokenResult{}, application_errors.ErrRefreshTokenGeneration
	}

	if strings.TrimSpace(newRefreshToken) == "" {
		return dto.RefreshTokenResult{}, application_errors.ErrRefreshTokenGeneration
	}

	// -------------------------------------------------------------------------
	// 7. Hash the replacement refresh token BEFORE revoking the old one.
	//
	// Only the hash will be persisted.
	// -------------------------------------------------------------------------

	newRefreshTokenHash, err := u.refreshTokenService.Hash(
		ctx,
		newRefreshToken,
	)
	if err != nil {
		return dto.RefreshTokenResult{}, application_errors.ErrRefreshTokenHashing
	}

	// -------------------------------------------------------------------------
	// 8. Generate the replacement access token BEFORE changing persistence.
	//
	// If access-token generation fails, the existing refresh session remains
	// usable.
	// -------------------------------------------------------------------------

	accessToken, err := u.tokenService.GenerateAccessToken(
		ctx,
		currentUser.ID(),
		currentUser.Role(),
	)
	if err != nil {
		return dto.RefreshTokenResult{}, application_errors.ErrTokenGeneration
	}

	now := time.Now()

	// -------------------------------------------------------------------------
	// 9. Revoke the old refresh-token session and persist the replacement.
	//
	// The repository's Rotate operation is responsible for performing this
	// lifecycle transition.
	// -------------------------------------------------------------------------

	newRecord := ports.RefreshTokenRecord{
		ID:        uuid.NewString(),
		UserID:    currentUser.ID().String(),
		TokenHash: newRefreshTokenHash,
		ExpiresAt: now.Add(30 * 24 * time.Hour),
		RevokedAt: nil,
		CreatedAt: now,
	}

	if err := u.refreshTokenRepository.Rotate(
		ctx,
		record.ID,
		now,
		newRecord,
	); err != nil {
		return dto.RefreshTokenResult{}, application_errors.ErrRefreshTokenPersistence
	}

	return dto.RefreshTokenResult{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

var _ ports.RefreshUserService = (*RefreshUserUseCase)(nil)
