package use_cases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// -----------------------------------------------------------------------------
// Fake Refresh Token Repository
// -----------------------------------------------------------------------------

type fakeRefreshTokenRepository struct {
	record *ports.RefreshTokenRecord

	findErr   error
	rotateErr error

	findCall   int
	rotateCall int

	rotatedOldTokenID string
	rotatedAt         time.Time
	rotatedRecord     ports.RefreshTokenRecord
}

// Create is retained because the repository port exposes it.
//
// The refresh use case should no longer call Create directly during token
// rotation. The method exists here so this fake continues to satisfy the
// complete repository port.
func (f *fakeRefreshTokenRepository) Create(
	_ context.Context,
	_ ports.RefreshTokenRecord,
) error {
	return nil
}

// FindByTokenHash returns the configured refresh-token session.
func (f *fakeRefreshTokenRepository) FindByTokenHash(
	_ context.Context,
	tokenHash string,
) (*ports.RefreshTokenRecord, error) {
	f.findCall++

	if f.findErr != nil {
		return nil, f.findErr
	}

	if f.record == nil {
		return nil, nil
	}

	if f.record.TokenHash != tokenHash {
		return nil, nil
	}

	return f.record, nil
}

// Revoke is retained because the repository port exposes it.
//
// The refresh use case should no longer call Revoke directly. Revocation is
// now part of the atomic Rotate operation.
func (f *fakeRefreshTokenRepository) Revoke(
	_ context.Context,
	_ string,
	_ time.Time,
) error {
	return nil
}

// Rotate records the complete atomic refresh-token rotation operation.
//
// The use case delegates both revocation of the old session and creation of
// the replacement session to this single repository operation.
func (f *fakeRefreshTokenRepository) Rotate(
	_ context.Context,
	oldTokenID string,
	revokedAt time.Time,
	newRecord ports.RefreshTokenRecord,
) error {
	f.rotateCall++

	f.rotatedOldTokenID = oldTokenID
	f.rotatedAt = revokedAt
	f.rotatedRecord = newRecord

	if f.rotateErr != nil {
		return f.rotateErr
	}

	return nil
}

var _ ports.RefreshTokenRepository = (*fakeRefreshTokenRepository)(nil)

// -----------------------------------------------------------------------------
// Fake Refresh Token Service
// -----------------------------------------------------------------------------

type fakeRefreshTokenService struct {
	generatedToken string
	generateErr    error

	hashedToken string
	hashErr     error

	generateCall int
	hashCall     int
	hashInput    string
}

func (f *fakeRefreshTokenService) Generate(
	_ context.Context,
) (string, error) {
	f.generateCall++

	if f.generateErr != nil {
		return "", f.generateErr
	}

	return f.generatedToken, nil
}

func (f *fakeRefreshTokenService) Hash(
	_ context.Context,
	token string,
) (string, error) {
	f.hashCall++
	f.hashInput = token

	if f.hashErr != nil {
		return "", f.hashErr
	}

	switch token {
	case "old-refresh-token":
		return "old-refresh-token-hash", nil

	case "refresh-token":
		return "token-hash", nil

	case "new-refresh-token":
		return "new-refresh-token-hash", nil

	default:
		return f.hashedToken, nil
	}
}

var _ ports.RefreshTokenService = (*fakeRefreshTokenService)(nil)

// -----------------------------------------------------------------------------
// Fake User Repository
// -----------------------------------------------------------------------------

type fakeRefreshUserRepository struct {
	user    *user.User
	findErr error

	findByIDCall int
}

func (f *fakeRefreshUserRepository) ExistsByEmail(
	_ context.Context,
	_ user.Email,
) (bool, error) {
	return false, nil
}

func (f *fakeRefreshUserRepository) Create(
	_ context.Context,
	_ *user.User,
) error {
	return nil
}

func (f *fakeRefreshUserRepository) FindByEmail(
	_ context.Context,
	_ user.Email,
) (*user.User, error) {
	return nil, nil
}

func (f *fakeRefreshUserRepository) FindByID(
	_ context.Context,
	id user.UserID,
) (*user.User, error) {
	f.findByIDCall++

	if f.findErr != nil {
		return nil, f.findErr
	}

	if f.user == nil {
		return nil, nil
	}

	if f.user.ID() != id {
		return nil, nil
	}

	return f.user, nil
}

var _ ports.UserRepository = (*fakeRefreshUserRepository)(nil)

// -----------------------------------------------------------------------------
// Fake Access Token Service
// -----------------------------------------------------------------------------

type fakeRefreshAccessTokenService struct {
	token       string
	generateErr error

	generateCall int
	userID       user.UserID
	role         user.Role
}

func (f *fakeRefreshAccessTokenService) GenerateAccessToken(
	_ context.Context,
	userID user.UserID,
	role user.Role,
) (string, error) {
	f.generateCall++

	f.userID = userID
	f.role = role

	if f.generateErr != nil {
		return "", f.generateErr
	}

	return f.token, nil
}

func (f *fakeRefreshAccessTokenService) ValidateAccessToken(
	_ context.Context,
	_ string,
) (ports.AuthenticatedIdentity, error) {
	return ports.AuthenticatedIdentity{}, nil
}

var _ ports.TokenService = (*fakeRefreshAccessTokenService)(nil)

// -----------------------------------------------------------------------------
// Test Helpers
// -----------------------------------------------------------------------------

func newRefreshTestUser(t *testing.T) *user.User {
	t.Helper()

	fullName, err := user.NewFullName("John Doe")
	if err != nil {
		t.Fatalf("failed to create full name: %v", err)
	}

	email, err := user.NewEmail("john@example.com")
	if err != nil {
		t.Fatalf("failed to create email: %v", err)
	}

	passwordHash, err := user.NewPasswordHash(
		"$2a$10$test-password-hash",
	)
	if err != nil {
		t.Fatalf("failed to create password hash: %v", err)
	}

	testUser, err := user.NewUser(
		fullName,
		email,
		passwordHash,
	)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	if err := testUser.Activate(); err != nil {
		t.Fatalf("failed to activate test user: %v", err)
	}

	return testUser
}

func newRefreshTokenRecord(
	testUser *user.User,
	tokenHash string,
	expiresAt time.Time,
) ports.RefreshTokenRecord {
	return ports.RefreshTokenRecord{
		ID:        "refresh-session-id",
		UserID:    testUser.ID().String(),
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		RevokedAt: nil,
		CreatedAt: time.Now().Add(-time.Hour),
	}
}

// -----------------------------------------------------------------------------
// Success
// -----------------------------------------------------------------------------

func TestRefreshUser_Success(t *testing.T) {
	t.Parallel()

	testUser := newRefreshTestUser(t)

	oldToken := "old-refresh-token"
	oldTokenHash := "old-refresh-token-hash"

	newToken := "new-refresh-token"
	newTokenHash := "new-refresh-token-hash"

	repository := &fakeRefreshTokenRepository{
		record: func() *ports.RefreshTokenRecord {
			record := newRefreshTokenRecord(
				testUser,
				oldTokenHash,
				time.Now().Add(time.Hour),
			)

			return &record
		}(),
	}

	refreshTokenService := &fakeRefreshTokenService{
		generatedToken: newToken,
		hashedToken:    newTokenHash,
	}

	userRepository := &fakeRefreshUserRepository{
		user: testUser,
	}

	tokenService := &fakeRefreshAccessTokenService{
		token: "new-access-token",
	}

	useCase := use_cases.NewRefreshUserUseCase(
		repository,
		userRepository,
		refreshTokenService,
		tokenService,
	)

	result, err := useCase.Refresh(
		context.Background(),
		dto.RefreshTokenCommand{
			RefreshToken: oldToken,
		},
	)

	if err != nil {
		t.Fatalf(
			"expected refresh to succeed, got error: %v",
			err,
		)
	}

	if result.AccessToken != "new-access-token" {
		t.Fatalf(
			"expected access token %q, got %q",
			"new-access-token",
			result.AccessToken,
		)
	}

	if result.RefreshToken != newToken {
		t.Fatalf(
			"expected refresh token %q, got %q",
			newToken,
			result.RefreshToken,
		)
	}

	if repository.findCall != 1 {
		t.Fatalf(
			"expected refresh-token lookup once, got %d",
			repository.findCall,
		)
	}

	if userRepository.findByIDCall != 1 {
		t.Fatalf(
			"expected user lookup once, got %d",
			userRepository.findByIDCall,
		)
	}

	if refreshTokenService.hashCall != 2 {
		t.Fatalf(
			"expected refresh token hashing twice, got %d",
			refreshTokenService.hashCall,
		)
	}

	if refreshTokenService.hashInput != newToken {
		t.Fatalf(
			"expected final hash input to be new token %q, got %q",
			newToken,
			refreshTokenService.hashInput,
		)
	}

	// The application layer must perform exactly one atomic rotation rather
	// than separately revoking and creating refresh-token records.
	if repository.rotateCall != 1 {
		t.Fatalf(
			"expected refresh-token rotation once, got %d",
			repository.rotateCall,
		)
	}

	if repository.rotatedOldTokenID != "refresh-session-id" {
		t.Fatalf(
			"expected rotated old session ID %q, got %q",
			"refresh-session-id",
			repository.rotatedOldTokenID,
		)
	}

	if repository.rotatedRecord.TokenHash != newTokenHash {
		t.Fatalf(
			"expected replacement token hash %q, got %q",
			newTokenHash,
			repository.rotatedRecord.TokenHash,
		)
	}

	if repository.rotatedRecord.UserID != testUser.ID().String() {
		t.Fatal(
			"expected replacement refresh session to belong to current user",
		)
	}

	if repository.rotatedRecord.RevokedAt != nil {
		t.Fatal(
			"expected replacement refresh token to be active",
		)
	}

	if repository.rotatedRecord.ID == "" {
		t.Fatal(
			"expected replacement refresh token to have a session ID",
		)
	}

	if repository.rotatedRecord.ExpiresAt.Before(time.Now()) {
		t.Fatal(
			"expected replacement refresh token to expire in the future",
		)
	}

	if tokenService.generateCall != 1 {
		t.Fatalf(
			"expected access token generation once, got %d",
			tokenService.generateCall,
		)
	}

	if tokenService.userID != testUser.ID() {
		t.Fatal("expected access token to use current user ID")
	}

	if tokenService.role != testUser.Role() {
		t.Fatal("expected access token to use current user role")
	}
}

// -----------------------------------------------------------------------------
// Invalid / Missing Refresh Token
// -----------------------------------------------------------------------------

func TestRefreshUser_InvalidRefreshToken(t *testing.T) {
	t.Parallel()

	repository := &fakeRefreshTokenRepository{}

	refreshTokenService := &fakeRefreshTokenService{
		hashedToken: "hash",
	}

	userRepository := &fakeRefreshUserRepository{}

	tokenService := &fakeRefreshAccessTokenService{
		token: "access-token",
	}

	useCase := use_cases.NewRefreshUserUseCase(
		repository,
		userRepository,
		refreshTokenService,
		tokenService,
	)

	_, err := useCase.Refresh(
		context.Background(),
		dto.RefreshTokenCommand{
			RefreshToken: "invalid-token",
		},
	)

	if !errors.Is(err, use_cases.ErrInvalidRefreshToken) {
		t.Fatalf(
			"expected ErrInvalidRefreshToken, got %v",
			err,
		)
	}

	if repository.findCall != 1 {
		t.Fatalf(
			"expected refresh-token lookup once, got %d",
			repository.findCall,
		)
	}

	if userRepository.findByIDCall != 0 {
		t.Fatal("expected user lookup not to occur")
	}

	if repository.rotateCall != 0 {
		t.Fatal("expected token not to be rotated")
	}

	if tokenService.generateCall != 0 {
		t.Fatal("expected access token not to be generated")
	}
}

// -----------------------------------------------------------------------------
// Revoked Refresh Token
// -----------------------------------------------------------------------------

func TestRefreshUser_RejectsRevokedToken(t *testing.T) {
	t.Parallel()

	testUser := newRefreshTestUser(t)

	revokedAt := time.Now().Add(-time.Minute)

	record := newRefreshTokenRecord(
		testUser,
		"token-hash",
		time.Now().Add(time.Hour),
	)
	record.RevokedAt = &revokedAt

	repository := &fakeRefreshTokenRepository{
		record: &record,
	}

	refreshTokenService := &fakeRefreshTokenService{
		hashedToken: "token-hash",
	}

	userRepository := &fakeRefreshUserRepository{
		user: testUser,
	}

	tokenService := &fakeRefreshAccessTokenService{
		token: "access-token",
	}

	useCase := use_cases.NewRefreshUserUseCase(
		repository,
		userRepository,
		refreshTokenService,
		tokenService,
	)

	_, err := useCase.Refresh(
		context.Background(),
		dto.RefreshTokenCommand{
			RefreshToken: "refresh-token",
		},
	)

	if !errors.Is(err, use_cases.ErrInvalidRefreshToken) {
		t.Fatalf(
			"expected ErrInvalidRefreshToken, got %v",
			err,
		)
	}

	if userRepository.findByIDCall != 0 {
		t.Fatal("expected revoked token not to load user")
	}

	if repository.rotateCall != 0 {
		t.Fatal("expected revoked token not to be rotated")
	}

	if tokenService.generateCall != 0 {
		t.Fatal("expected access token not to be generated")
	}
}

// -----------------------------------------------------------------------------
// Expired Refresh Token
// -----------------------------------------------------------------------------

func TestRefreshUser_RejectsExpiredToken(t *testing.T) {
	t.Parallel()

	testUser := newRefreshTestUser(t)

	record := newRefreshTokenRecord(
		testUser,
		"token-hash",
		time.Now().Add(-time.Minute),
	)

	repository := &fakeRefreshTokenRepository{
		record: &record,
	}

	refreshTokenService := &fakeRefreshTokenService{
		hashedToken: "token-hash",
	}

	userRepository := &fakeRefreshUserRepository{
		user: testUser,
	}

	tokenService := &fakeRefreshAccessTokenService{
		token: "access-token",
	}

	useCase := use_cases.NewRefreshUserUseCase(
		repository,
		userRepository,
		refreshTokenService,
		tokenService,
	)

	_, err := useCase.Refresh(
		context.Background(),
		dto.RefreshTokenCommand{
			RefreshToken: "refresh-token",
		},
	)

	if !errors.Is(err, use_cases.ErrInvalidRefreshToken) {
		t.Fatalf(
			"expected ErrInvalidRefreshToken, got %v",
			err,
		)
	}

	if userRepository.findByIDCall != 0 {
		t.Fatal("expected expired token not to load user")
	}

	if repository.rotateCall != 0 {
		t.Fatal("expected expired token not to be rotated")
	}

	if tokenService.generateCall != 0 {
		t.Fatal("expected access token not to be generated")
	}
}

// -----------------------------------------------------------------------------
// User Not Found
// -----------------------------------------------------------------------------

func TestRefreshUser_UserNotFound(t *testing.T) {
	t.Parallel()

	record := ports.RefreshTokenRecord{
		ID:        "refresh-session-id",
		UserID:    user.NewUserID().String(),
		TokenHash: "token-hash",
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}

	repository := &fakeRefreshTokenRepository{
		record: &record,
	}

	refreshTokenService := &fakeRefreshTokenService{
		hashedToken: "token-hash",
	}

	userRepository := &fakeRefreshUserRepository{}

	tokenService := &fakeRefreshAccessTokenService{
		token: "access-token",
	}

	useCase := use_cases.NewRefreshUserUseCase(
		repository,
		userRepository,
		refreshTokenService,
		tokenService,
	)

	_, err := useCase.Refresh(
		context.Background(),
		dto.RefreshTokenCommand{
			RefreshToken: "refresh-token",
		},
	)

	if !errors.Is(err, use_cases.ErrInvalidRefreshToken) {
		t.Fatalf(
			"expected ErrInvalidRefreshToken, got %v",
			err,
		)
	}

	if repository.rotateCall != 0 {
		t.Fatal("expected token not to be rotated")
	}

	if tokenService.generateCall != 0 {
		t.Fatal("expected access token not to be generated")
	}
}

// -----------------------------------------------------------------------------
// Inactive User
// -----------------------------------------------------------------------------

func TestRefreshUser_RejectsInactiveUser(t *testing.T) {
	t.Parallel()

	testUser := newRefreshTestUser(t)

	if err := testUser.Deactivate(); err != nil {
		t.Fatalf("failed to deactivate test user: %v", err)
	}

	record := newRefreshTokenRecord(
		testUser,
		"token-hash",
		time.Now().Add(time.Hour),
	)

	repository := &fakeRefreshTokenRepository{
		record: &record,
	}

	refreshTokenService := &fakeRefreshTokenService{
		hashedToken: "token-hash",
	}

	userRepository := &fakeRefreshUserRepository{
		user: testUser,
	}

	tokenService := &fakeRefreshAccessTokenService{
		token: "access-token",
	}

	useCase := use_cases.NewRefreshUserUseCase(
		repository,
		userRepository,
		refreshTokenService,
		tokenService,
	)

	_, err := useCase.Refresh(
		context.Background(),
		dto.RefreshTokenCommand{
			RefreshToken: "refresh-token",
		},
	)

	if !errors.Is(err, use_cases.ErrInvalidRefreshToken) {
		t.Fatalf(
			"expected ErrInvalidRefreshToken, got %v",
			err,
		)
	}

	if repository.rotateCall != 0 {
		t.Fatal("expected inactive user's token not to be rotated")
	}

	if tokenService.generateCall != 0 {
		t.Fatal("expected access token not to be generated")
	}
}

// -----------------------------------------------------------------------------
// Hash Failure
// -----------------------------------------------------------------------------

func TestRefreshUser_HashFailure(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("hashing failed")

	repository := &fakeRefreshTokenRepository{}

	refreshTokenService := &fakeRefreshTokenService{
		hashErr: expectedErr,
	}

	userRepository := &fakeRefreshUserRepository{}

	tokenService := &fakeRefreshAccessTokenService{}

	useCase := use_cases.NewRefreshUserUseCase(
		repository,
		userRepository,
		refreshTokenService,
		tokenService,
	)

	_, err := useCase.Refresh(
		context.Background(),
		dto.RefreshTokenCommand{
			RefreshToken: "refresh-token",
		},
	)

	if !errors.Is(err, use_cases.ErrRefreshTokenHashing) {
		t.Fatalf(
			"expected ErrRefreshTokenHashing, got %v",
			err,
		)
	}

	if repository.findCall != 0 {
		t.Fatal("expected repository lookup not to occur after hash failure")
	}
}

// -----------------------------------------------------------------------------
// Rotation Failure
// -----------------------------------------------------------------------------

func TestRefreshUser_RotationFailure(t *testing.T) {
	t.Parallel()

	testUser := newRefreshTestUser(t)

	expectedErr := errors.New("rotation failed")

	record := newRefreshTokenRecord(
		testUser,
		"token-hash",
		time.Now().Add(time.Hour),
	)

	repository := &fakeRefreshTokenRepository{
		record:    &record,
		rotateErr: expectedErr,
	}

	refreshTokenService := &fakeRefreshTokenService{
		generatedToken: "new-token",
		hashedToken:    "new-token-hash",
	}

	userRepository := &fakeRefreshUserRepository{
		user: testUser,
	}

	tokenService := &fakeRefreshAccessTokenService{
		token: "access-token",
	}

	useCase := use_cases.NewRefreshUserUseCase(
		repository,
		userRepository,
		refreshTokenService,
		tokenService,
	)

	_, err := useCase.Refresh(
		context.Background(),
		dto.RefreshTokenCommand{
			RefreshToken: "refresh-token",
		},
	)

	if !errors.Is(err, use_cases.ErrRefreshTokenPersistence) {
		t.Fatalf(
			"expected ErrRefreshTokenPersistence, got %v",
			err,
		)
	}

	if repository.rotateCall != 1 {
		t.Fatalf(
			"expected Rotate() to be called once, got %d",
			repository.rotateCall,
		)
	}

	if repository.rotatedOldTokenID != record.ID {
		t.Fatalf(
			"expected Rotate() to receive old session ID %q, got %q",
			record.ID,
			repository.rotatedOldTokenID,
		)
	}

	if repository.rotatedRecord.TokenHash != "new-token-hash" {
		t.Fatalf(
			"expected Rotate() to receive new token hash %q, got %q",
			"new-token-hash",
			repository.rotatedRecord.TokenHash,
		)
	}

	if tokenService.generateCall != 1 {
		t.Fatalf(
			"expected access token to be generated once before persistence rotation, got %d",
			tokenService.generateCall,
		)
	}
}

// -----------------------------------------------------------------------------
// New Refresh Token Generation Failure
// -----------------------------------------------------------------------------

func TestRefreshUser_RefreshTokenGenerationFailure(t *testing.T) {
	t.Parallel()

	testUser := newRefreshTestUser(t)

	record := newRefreshTokenRecord(
		testUser,
		"token-hash",
		time.Now().Add(time.Hour),
	)

	repository := &fakeRefreshTokenRepository{
		record: &record,
	}

	refreshTokenService := &fakeRefreshTokenService{
		generateErr: errors.New("generation failed"),
	}

	userRepository := &fakeRefreshUserRepository{
		user: testUser,
	}

	tokenService := &fakeRefreshAccessTokenService{
		token: "access-token",
	}

	useCase := use_cases.NewRefreshUserUseCase(
		repository,
		userRepository,
		refreshTokenService,
		tokenService,
	)

	_, err := useCase.Refresh(
		context.Background(),
		dto.RefreshTokenCommand{
			RefreshToken: "refresh-token",
		},
	)

	if !errors.Is(err, use_cases.ErrRefreshTokenGeneration) {
		t.Fatalf(
			"expected ErrRefreshTokenGeneration, got %v",
			err,
		)
	}

	if repository.rotateCall != 0 {
		t.Fatal(
			"expected old refresh token not to be rotated when new token generation fails",
		)
	}

	if tokenService.generateCall != 0 {
		t.Fatal(
			"expected access token not to be generated when refresh token generation fails",
		)
	}
}

// -----------------------------------------------------------------------------
// New Refresh Token Hash Failure
// -----------------------------------------------------------------------------

func TestRefreshUser_NewRefreshTokenHashFailure(t *testing.T) {
	t.Parallel()

	testUser := newRefreshTestUser(t)

	record := newRefreshTokenRecord(
		testUser,
		"token-hash",
		time.Now().Add(time.Hour),
	)

	repository := &fakeRefreshTokenRepository{
		record: &record,
	}

	refreshTokenService := &fakeRefreshTokenService{
		generatedToken: "new-refresh-token",
		hashErr:        errors.New("hash failed"),
	}

	userRepository := &fakeRefreshUserRepository{
		user: testUser,
	}

	tokenService := &fakeRefreshAccessTokenService{
		token: "access-token",
	}

	useCase := use_cases.NewRefreshUserUseCase(
		repository,
		userRepository,
		refreshTokenService,
		tokenService,
	)

	_, err := useCase.Refresh(
		context.Background(),
		dto.RefreshTokenCommand{
			RefreshToken: "refresh-token",
		},
	)

	if !errors.Is(err, use_cases.ErrRefreshTokenHashing) {
		t.Fatalf(
			"expected ErrRefreshTokenHashing, got %v",
			err,
		)
	}

	if repository.rotateCall != 0 {
		t.Fatal(
			"expected old refresh token not to be rotated when new token hashing fails",
		)
	}

	if tokenService.generateCall != 0 {
		t.Fatal(
			"expected access token not to be generated when refresh token hashing fails",
		)
	}
}

// -----------------------------------------------------------------------------
// Access Token Generation Failure
// -----------------------------------------------------------------------------

func TestRefreshUser_AccessTokenGenerationFailure(t *testing.T) {
	t.Parallel()

	testUser := newRefreshTestUser(t)

	record := newRefreshTokenRecord(
		testUser,
		"token-hash",
		time.Now().Add(time.Hour),
	)

	repository := &fakeRefreshTokenRepository{
		record: &record,
	}

	refreshTokenService := &fakeRefreshTokenService{
		generatedToken: "new-refresh-token",
		hashedToken:    "new-refresh-token-hash",
	}

	userRepository := &fakeRefreshUserRepository{
		user: testUser,
	}

	tokenService := &fakeRefreshAccessTokenService{
		generateErr: errors.New("access token generation failed"),
	}

	useCase := use_cases.NewRefreshUserUseCase(
		repository,
		userRepository,
		refreshTokenService,
		tokenService,
	)

	_, err := useCase.Refresh(
		context.Background(),
		dto.RefreshTokenCommand{
			RefreshToken: "refresh-token",
		},
	)

	if !errors.Is(err, use_cases.ErrTokenGeneration) {
		t.Fatalf(
			"expected ErrTokenGeneration, got %v",
			err,
		)
	}

	if repository.rotateCall != 0 {
		t.Fatal(
			"expected old refresh token not to be rotated when access token generation fails",
		)
	}
}
