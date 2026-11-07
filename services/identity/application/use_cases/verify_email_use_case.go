package use_cases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
)

// VerifyEmailUseCase consumes an activation token and activates its account.
type VerifyEmailUseCase struct {
	tokenRepository ports.ActivationTokenRepository
}

func NewVerifyEmailUseCase(
	tokenRepository ports.ActivationTokenRepository,
) *VerifyEmailUseCase {
	return &VerifyEmailUseCase{
		tokenRepository: tokenRepository,
	}
}

func (uc *VerifyEmailUseCase) Execute(
	ctx context.Context,
	rawToken string,
) error {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return application_errors.ErrInvalidActivationToken
	}

	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	return uc.tokenRepository.ConsumeAndActivate(
		ctx,
		tokenHash,
		time.Now().UTC(),
	)
}
