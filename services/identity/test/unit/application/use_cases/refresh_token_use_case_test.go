package use_cases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
)

type mockRefreshTokenService struct {
	generatedToken string
	generateErr    error

	hashedToken string
	hashErr     error
}

func (m *mockRefreshTokenService) Generate(
	ctx context.Context,
) (string, error) {
	if m.generateErr != nil {
		return "", m.generateErr
	}

	return m.generatedToken, nil
}

func (m *mockRefreshTokenService) Hash(
	ctx context.Context,
	token string,
) (string, error) {
	if m.hashErr != nil {
		return "", m.hashErr
	}

	return m.hashedToken, nil
}

type mockRefreshTokenRepository struct {
	record *ports.RefreshTokenRecord

	createErr error
	findErr   error
	revokeErr error

	createdRecord *ports.RefreshTokenRecord
	revokedID     string
}

func (m *mockRefreshTokenRepository) Create(
	ctx context.Context,
	record ports.RefreshTokenRecord,
) error {
	if m.createErr != nil {
		return m.createErr
	}

	recordCopy := record
	m.createdRecord = &recordCopy

	return nil
}

func (m *mockRefreshTokenRepository) FindByTokenHash(
	ctx context.Context,
	tokenHash string,
) (*ports.RefreshTokenRecord, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}

	return m.record, nil
}

func (m *mockRefreshTokenRepository) Revoke(
	ctx context.Context,
	id string,
	revokedAt time.Time,
) error {
	if m.revokeErr != nil {
		return m.revokeErr
	}

	m.revokedID = id

	return nil
}

func (m *mockRefreshTokenRepository) Rotate(
	ctx context.Context,
	oldTokenID string,
	revokedAt time.Time,
	newRecord ports.RefreshTokenRecord,
) error {
	if m.revokeErr != nil {
		return m.revokeErr
	}

	m.revokedID = oldTokenID

	recordCopy := newRecord
	m.createdRecord = &recordCopy

	return nil
}

type mockAccessTokenService struct {
	accessToken string
	generateErr error
}

func (m *mockAccessTokenService) GenerateAccessToken(
	ctx context.Context,
	userID interface{},
	role interface{},
) (string, error) {
	return m.accessToken, m.generateErr
}

func TestRefreshTokenUseCase_Refresh_Success(t *testing.T) {
	t.Parallel()

	now := time.Now()

	refreshTokenService := &mockRefreshTokenService{
		hashedToken:    "hashed-old-token",
		generatedToken: "new-refresh-token",
	}

	repository := &mockRefreshTokenRepository{
		record: &ports.RefreshTokenRecord{
			ID:        "session-1",
			UserID:    "user-1",
			TokenHash: "hashed-old-token",
			ExpiresAt: now.Add(time.Hour),
			CreatedAt: now.Add(-time.Hour),
		},
	}

	_ = refreshTokenService
	_ = repository
	_ = dto.RefreshTokenCommand{}
	_ = use_cases.ErrInvalidRefreshToken
	_ = errors.New
}
