package use_cases_test

import (
	"context"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
)

// -----------------------------------------------------------------------------
// Shared Refresh Token Service Mock
// -----------------------------------------------------------------------------
//
// This mock is shared by the logout use-case tests.
//
// The refresh-user tests intentionally use their own
// fakeRefreshTokenService because those tests need additional call tracking
// and token-specific behavior.
//
// Keeping this mock separate prevents the deletion of the obsolete
// refresh_token_use_case_test.go file from breaking logout tests.

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

var _ ports.RefreshTokenService = (*mockRefreshTokenService)(nil)

// -----------------------------------------------------------------------------
// Shared Refresh Token Repository Mock
// -----------------------------------------------------------------------------
//
// This mock is shared by the logout use-case tests.
//
// It implements the complete RefreshTokenRepository interface, including
// Create and Rotate, even though logout only needs FindByTokenHash and Revoke.

type mockRefreshTokenRepository struct {
	record *ports.RefreshTokenRecord

	createErr error
	findErr   error
	revokeErr error
	rotateErr error

	createdRecord *ports.RefreshTokenRecord

	revokedID string
	revokedAt time.Time

	rotatedOldTokenID string
	rotatedAt         time.Time
	rotatedRecord     ports.RefreshTokenRecord
}

func (m *mockRefreshTokenRepository) Create(
	ctx context.Context,
	record ports.RefreshTokenRecord,
) error {
	if m.createErr != nil {
		return m.createErr
	}

	copiedRecord := record
	m.createdRecord = &copiedRecord

	return nil
}

func (m *mockRefreshTokenRepository) FindByTokenHash(
	ctx context.Context,
	tokenHash string,
) (*ports.RefreshTokenRecord, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}

	if m.record == nil {
		return nil, nil
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
	m.revokedAt = revokedAt

	return nil
}

func (m *mockRefreshTokenRepository) Rotate(
	ctx context.Context,
	oldTokenID string,
	revokedAt time.Time,
	newRecord ports.RefreshTokenRecord,
) error {
	if m.rotateErr != nil {
		return m.rotateErr
	}

	m.rotatedOldTokenID = oldTokenID
	m.rotatedAt = revokedAt
	m.rotatedRecord = newRecord

	return nil
}

var _ ports.RefreshTokenRepository = (*mockRefreshTokenRepository)(nil)
