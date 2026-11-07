package use_cases

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"

	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/activation"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// RequestActivationUseCase issues an activation token and emails its link.
type RequestActivationUseCase struct {
	userRepository  ports.UserRepository
	tokenRepository ports.ActivationTokenRepository
	emailSender     ports.ActivationEmailSender
	activationURL   string
	tokenTTL        time.Duration
}

func NewRequestActivationUseCase(
	userRepository ports.UserRepository,
	tokenRepository ports.ActivationTokenRepository,
	emailSender ports.ActivationEmailSender,
	activationURL string,
	tokenTTL time.Duration,
) *RequestActivationUseCase {
	return &RequestActivationUseCase{
		userRepository:  userRepository,
		tokenRepository: tokenRepository,
		emailSender:     emailSender,
		activationURL:   activationURL,
		tokenTTL:        tokenTTL,
	}
}

// Execute requests activation without revealing whether an email is registered.
// Only pending-verification accounts receive an activation email.
func (uc *RequestActivationUseCase) Execute(
	ctx context.Context,
	emailAddress string,
) error {
	email, err := user.NewEmail(emailAddress)
	if err != nil {
		// Keep the public request response independent of account existence.
		return nil
	}

	existingUser, err := uc.userRepository.FindByEmail(ctx, email)
	if err != nil {
		return err
	}
	if existingUser == nil || existingUser.Status() != user.StatusPendingVerification {
		return nil
	}

	if uc.tokenTTL <= 0 {
		return application_errors.ErrInvalidActivationURL
	}

	parsedURL, err := url.Parse(strings.TrimSpace(uc.activationURL))
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return application_errors.ErrInvalidActivationURL
	}

	rawToken, err := generateActivationToken()
	if err != nil {
		return application_errors.ErrActivationTokenGeneration
	}

	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	now := time.Now().UTC()
	token, err := activation.NewToken(
		existingUser.ID(),
		tokenHash,
		now.Add(uc.tokenTTL),
	)
	if err != nil {
		return err
	}

	// Replacing previous tokens ensures only the latest link remains valid.
	if err := uc.tokenRepository.DeleteByUserID(ctx, existingUser.ID()); err != nil {
		return err
	}

	if err := uc.tokenRepository.Create(ctx, token); err != nil {
		return err
	}

	query := parsedURL.Query()
	query.Set("token", rawToken)
	parsedURL.RawQuery = query.Encode()

	if err := uc.emailSender.SendActivationEmail(
		ctx,
		existingUser.Email(),
		parsedURL.String(),
	); err != nil {
		// Do not leave a valid token behind when delivery fails.
		_ = uc.tokenRepository.DeleteByUserID(ctx, existingUser.ID())
		return fmt.Errorf("%w: %v",
			application_errors.ErrActivationEmailDelivery,
			err,
		)
	}

	return nil
}

func generateActivationToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}
