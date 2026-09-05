package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// -----------------------------------------------------------------------------
// Fake User Repository
// -----------------------------------------------------------------------------

// fakeAuthenticationUserRepository is a test double for UserRepository.
//
// Authentication primarily uses FindByEmail. The other methods are
// implemented because they are part of the shared repository contract.
type fakeAuthenticationUserRepository struct {
	user     *user.User
	findErr  error
	findCall int
}

func (f *fakeAuthenticationUserRepository) ExistsByEmail(
	ctx context.Context,
	email user.Email,
) (bool, error) {
	if f.user == nil {
		return false, nil
	}

	return f.user.Email().String() == email.String(), nil
}

func (f *fakeAuthenticationUserRepository) Create(
	ctx context.Context,
	newUser *user.User,
) error {
	return nil
}

// FindByEmail returns the configured user when the email matches.
func (f *fakeAuthenticationUserRepository) FindByEmail(
	ctx context.Context,
	email user.Email,
) (*user.User, error) {
	f.findCall++

	if f.findErr != nil {
		return nil, f.findErr
	}

	if f.user == nil {
		return nil, nil
	}

	if f.user.Email().String() != email.String() {
		return nil, nil
	}

	return f.user, nil
}

// FindByID returns the configured user when the ID matches.
func (f *fakeAuthenticationUserRepository) FindByID(
	ctx context.Context,
	id user.UserID,
) (*user.User, error) {
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

// -----------------------------------------------------------------------------
// Fake Password Hasher
// -----------------------------------------------------------------------------

// fakeAuthenticationPasswordHasher is a test double for PasswordHasher.
//
// It allows authentication tests to control whether password verification
// succeeds without depending on the concrete bcrypt implementation.
type fakeAuthenticationPasswordHasher struct {
	verifyErr  error
	verifyCall int

	plainPassword string
	passwordHash  string
}

func (f *fakeAuthenticationPasswordHasher) Hash(
	ctx context.Context,
	plainPassword string,
) (string, error) {
	return "test-password-hash", nil
}

// Verify records the credentials supplied by the use case.
func (f *fakeAuthenticationPasswordHasher) Verify(
	ctx context.Context,
	plainPassword string,
	passwordHash string,
) error {
	f.verifyCall++

	f.plainPassword = plainPassword
	f.passwordHash = passwordHash

	return f.verifyErr
}

// -----------------------------------------------------------------------------
// Fake Token Service
// -----------------------------------------------------------------------------

// fakeAuthenticationTokenService is a test double for TokenService.
//
// It allows us to verify that access-token generation occurs only after
// successful authentication.
type fakeAuthenticationTokenService struct {
	token       string
	generateErr error

	generateCall int
	userID       user.UserID
	role         user.Role
}

func (f *fakeAuthenticationTokenService) GenerateAccessToken(
	ctx context.Context,
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

func (f *fakeAuthenticationTokenService) ValidateAccessToken(
	ctx context.Context,
	token string,
) (ports.AuthenticatedIdentity, error) {
	return ports.AuthenticatedIdentity{}, nil
}

// -----------------------------------------------------------------------------
// Compile-Time Contract Assertions
// -----------------------------------------------------------------------------

var _ ports.UserRepository = (*fakeAuthenticationUserRepository)(nil)

var _ ports.PasswordHasher = (*fakeAuthenticationPasswordHasher)(nil)

var _ ports.TokenService = (*fakeAuthenticationTokenService)(nil)

// -----------------------------------------------------------------------------
// Test Helpers
// -----------------------------------------------------------------------------

// newAuthenticationUser creates an active user suitable for authentication
// tests.
func newAuthenticationUser(t *testing.T) *user.User {
	t.Helper()

	fullName, err := user.NewFullName("John Doe")
	if err != nil {
		t.Fatalf("failed to create full name: %v", err)
	}

	email, err := user.NewEmail("john@example.com")
	if err != nil {
		t.Fatalf("failed to create email: %v", err)
	}

	passwordHash, err := user.NewPasswordHash("$2a$10$test-password-hash")
	if err != nil {
		t.Fatalf("failed to create password hash: %v", err)
	}

	authenticatedUser, err := user.NewUser(
		fullName,
		email,
		passwordHash,
	)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Newly created users start in pending verification.
	// Authentication tests require an active account.
	if err := authenticatedUser.Activate(); err != nil {
		t.Fatalf("failed to activate test user: %v", err)
	}

	return authenticatedUser
}

// newPendingAuthenticationUser creates a user that remains in the
// pending-verification state.
func newPendingAuthenticationUser(t *testing.T) *user.User {
	t.Helper()

	fullName, err := user.NewFullName("Pending User")
	if err != nil {
		t.Fatalf("failed to create full name: %v", err)
	}

	email, err := user.NewEmail("pending@example.com")
	if err != nil {
		t.Fatalf("failed to create email: %v", err)
	}

	passwordHash, err := user.NewPasswordHash("$2a$10$test-password-hash")
	if err != nil {
		t.Fatalf("failed to create password hash: %v", err)
	}

	pendingUser, err := user.NewUser(
		fullName,
		email,
		passwordHash,
	)
	if err != nil {
		t.Fatalf("failed to create pending test user: %v", err)
	}

	return pendingUser
}

// newAuthenticationUseCase creates the authentication use case.
func newAuthenticationUseCase(
	t *testing.T,
	repository *fakeAuthenticationUserRepository,
	passwordHasher *fakeAuthenticationPasswordHasher,
	tokenService *fakeAuthenticationTokenService,
) *use_cases.AuthenticateUserUseCase {
	t.Helper()

	return use_cases.NewAuthenticateUserUseCase(
		repository,
		passwordHasher,
		tokenService,
	)
}

// -----------------------------------------------------------------------------
// Success
// -----------------------------------------------------------------------------

func TestAuthenticateUser_Success(t *testing.T) {
	testUser := newAuthenticationUser(t)

	repository := &fakeAuthenticationUserRepository{
		user: testUser,
	}

	passwordHasher := &fakeAuthenticationPasswordHasher{}

	tokenService := &fakeAuthenticationTokenService{
		token: "test-access-token",
	}

	useCase := newAuthenticationUseCase(
		t,
		repository,
		passwordHasher,
		tokenService,
	)

	command := dto.LoginUserCommand{
		Email:    testUser.Email().String(),
		Password: "correct-password",
	}

	result, err := useCase.Authenticate(
		context.Background(),
		command,
	)
	if err != nil {
		t.Fatalf("expected authentication to succeed, got error: %v", err)
	}

	if result.ID != testUser.ID().String() {
		t.Fatalf(
			"expected ID %q, got %q",
			testUser.ID().String(),
			result.ID,
		)
	}

	if result.Email != testUser.Email().String() {
		t.Fatalf(
			"expected email %q, got %q",
			testUser.Email().String(),
			result.Email,
		)
	}

	if result.Role != testUser.Role().String() {
		t.Fatalf(
			"expected role %q, got %q",
			testUser.Role().String(),
			result.Role,
		)
	}

	if result.Status != testUser.Status().String() {
		t.Fatalf(
			"expected status %q, got %q",
			testUser.Status().String(),
			result.Status,
		)
	}

	if result.AccessToken != "test-access-token" {
		t.Fatalf(
			"expected access token %q, got %q",
			"test-access-token",
			result.AccessToken,
		)
	}

	if passwordHasher.verifyCall != 1 {
		t.Fatalf(
			"expected password verification to be called once, got %d",
			passwordHasher.verifyCall,
		)
	}

	if passwordHasher.plainPassword != "correct-password" {
		t.Fatalf(
			"expected plaintext password %q, got %q",
			"correct-password",
			passwordHasher.plainPassword,
		)
	}

	if passwordHasher.passwordHash != testUser.PasswordHash().String() {
		t.Fatal("expected persisted password hash to be verified")
	}

	if tokenService.generateCall != 1 {
		t.Fatalf(
			"expected token generation to be called once, got %d",
			tokenService.generateCall,
		)
	}

	if tokenService.userID != testUser.ID() {
		t.Fatal("expected token service to receive authenticated user ID")
	}

	if tokenService.role != testUser.Role() {
		t.Fatal("expected token service to receive authenticated user role")
	}
}

// -----------------------------------------------------------------------------
// User Not Found
// -----------------------------------------------------------------------------

func TestAuthenticateUser_UserNotFound(t *testing.T) {
	repository := &fakeAuthenticationUserRepository{}

	passwordHasher := &fakeAuthenticationPasswordHasher{}

	tokenService := &fakeAuthenticationTokenService{
		token: "test-access-token",
	}

	useCase := newAuthenticationUseCase(
		t,
		repository,
		passwordHasher,
		tokenService,
	)

	command := dto.LoginUserCommand{
		Email:    "missing@example.com",
		Password: "correct-password",
	}

	_, err := useCase.Authenticate(
		context.Background(),
		command,
	)

	if !errors.Is(err, use_cases.ErrInvalidCredentials) {
		t.Fatalf(
			"expected ErrInvalidCredentials, got %v",
			err,
		)
	}

	// Password verification must never occur when the user does not exist.
	if passwordHasher.verifyCall != 0 {
		t.Fatalf(
			"expected password verification not to be called, got %d calls",
			passwordHasher.verifyCall,
		)
	}

	// Token generation must never occur when authentication fails.
	if tokenService.generateCall != 0 {
		t.Fatalf(
			"expected token generation not to be called, got %d calls",
			tokenService.generateCall,
		)
	}
}

// -----------------------------------------------------------------------------
// Invalid Password
// -----------------------------------------------------------------------------

func TestAuthenticateUser_InvalidPassword(t *testing.T) {
	testUser := newAuthenticationUser(t)

	repository := &fakeAuthenticationUserRepository{
		user: testUser,
	}

	passwordHasher := &fakeAuthenticationPasswordHasher{
		verifyErr: errors.New("password does not match"),
	}

	tokenService := &fakeAuthenticationTokenService{
		token: "test-access-token",
	}

	useCase := newAuthenticationUseCase(
		t,
		repository,
		passwordHasher,
		tokenService,
	)

	command := dto.LoginUserCommand{
		Email:    testUser.Email().String(),
		Password: "wrong-password",
	}

	_, err := useCase.Authenticate(
		context.Background(),
		command,
	)

	if !errors.Is(err, use_cases.ErrInvalidCredentials) {
		t.Fatalf(
			"expected ErrInvalidCredentials, got %v",
			err,
		)
	}

	if passwordHasher.verifyCall != 1 {
		t.Fatalf(
			"expected password verification to be called once, got %d",
			passwordHasher.verifyCall,
		)
	}

	if tokenService.generateCall != 0 {
		t.Fatalf(
			"expected token generation not to be called, got %d calls",
			tokenService.generateCall,
		)
	}
}

// -----------------------------------------------------------------------------
// Inactive Account
// -----------------------------------------------------------------------------

func TestAuthenticateUser_InactiveAccount(t *testing.T) {
	testUser := newAuthenticationUser(t)

	if err := testUser.Deactivate(); err != nil {
		t.Fatalf("failed to deactivate test user: %v", err)
	}

	repository := &fakeAuthenticationUserRepository{
		user: testUser,
	}

	passwordHasher := &fakeAuthenticationPasswordHasher{}

	tokenService := &fakeAuthenticationTokenService{
		token: "test-access-token",
	}

	useCase := newAuthenticationUseCase(
		t,
		repository,
		passwordHasher,
		tokenService,
	)

	command := dto.LoginUserCommand{
		Email:    testUser.Email().String(),
		Password: "correct-password",
	}

	_, err := useCase.Authenticate(
		context.Background(),
		command,
	)

	if !errors.Is(err, use_cases.ErrAccountNotActive) {
		t.Fatalf(
			"expected ErrAccountNotActive, got %v",
			err,
		)
	}

	if passwordHasher.verifyCall != 1 {
		t.Fatalf(
			"expected password verification to be called once, got %d",
			passwordHasher.verifyCall,
		)
	}

	if tokenService.generateCall != 0 {
		t.Fatalf(
			"expected token generation not to be called, got %d calls",
			tokenService.generateCall,
		)
	}
}

// -----------------------------------------------------------------------------
// Pending Verification Account
// -----------------------------------------------------------------------------

func TestAuthenticateUser_PendingVerification(t *testing.T) {
	pendingUser := newPendingAuthenticationUser(t)

	repository := &fakeAuthenticationUserRepository{
		user: pendingUser,
	}

	passwordHasher := &fakeAuthenticationPasswordHasher{}

	tokenService := &fakeAuthenticationTokenService{
		token: "test-access-token",
	}

	useCase := newAuthenticationUseCase(
		t,
		repository,
		passwordHasher,
		tokenService,
	)

	command := dto.LoginUserCommand{
		Email:    pendingUser.Email().String(),
		Password: "correct-password",
	}

	_, err := useCase.Authenticate(
		context.Background(),
		command,
	)

	if !errors.Is(err, use_cases.ErrAccountNotActive) {
		t.Fatalf(
			"expected ErrAccountNotActive, got %v",
			err,
		)
	}

	if passwordHasher.verifyCall != 1 {
		t.Fatalf(
			"expected password verification to be called once, got %d",
			passwordHasher.verifyCall,
		)
	}

	if tokenService.generateCall != 0 {
		t.Fatalf(
			"expected token generation not to be called, got %d calls",
			tokenService.generateCall,
		)
	}
}

// -----------------------------------------------------------------------------
// Suspended Account
// -----------------------------------------------------------------------------

func TestAuthenticateUser_SuspendedAccount(t *testing.T) {
	testUser := newAuthenticationUser(t)

	if err := testUser.Suspend(); err != nil {
		t.Fatalf("failed to suspend test user: %v", err)
	}

	repository := &fakeAuthenticationUserRepository{
		user: testUser,
	}

	passwordHasher := &fakeAuthenticationPasswordHasher{}

	tokenService := &fakeAuthenticationTokenService{
		token: "test-access-token",
	}

	useCase := newAuthenticationUseCase(
		t,
		repository,
		passwordHasher,
		tokenService,
	)

	command := dto.LoginUserCommand{
		Email:    testUser.Email().String(),
		Password: "correct-password",
	}

	_, err := useCase.Authenticate(
		context.Background(),
		command,
	)

	if !errors.Is(err, use_cases.ErrAccountNotActive) {
		t.Fatalf(
			"expected ErrAccountNotActive, got %v",
			err,
		)
	}

	if passwordHasher.verifyCall != 1 {
		t.Fatalf(
			"expected password verification to be called once, got %d",
			passwordHasher.verifyCall,
		)
	}

	if tokenService.generateCall != 0 {
		t.Fatalf(
			"expected token generation not to be called, got %d calls",
			tokenService.generateCall,
		)
	}
}

// -----------------------------------------------------------------------------
// Invalid Email
// -----------------------------------------------------------------------------

func TestAuthenticateUser_InvalidEmail(t *testing.T) {
	repository := &fakeAuthenticationUserRepository{}

	passwordHasher := &fakeAuthenticationPasswordHasher{}

	tokenService := &fakeAuthenticationTokenService{
		token: "test-access-token",
	}

	useCase := newAuthenticationUseCase(
		t,
		repository,
		passwordHasher,
		tokenService,
	)

	command := dto.LoginUserCommand{
		Email:    "not-an-email",
		Password: "correct-password",
	}

	_, err := useCase.Authenticate(
		context.Background(),
		command,
	)

	if !errors.Is(err, use_cases.ErrInvalidCredentials) {
		t.Fatalf(
			"expected ErrInvalidCredentials, got %v",
			err,
		)
	}

	if repository.findCall != 0 {
		t.Fatalf(
			"expected repository lookup not to occur, got %d calls",
			repository.findCall,
		)
	}

	if passwordHasher.verifyCall != 0 {
		t.Fatalf(
			"expected password verification not to occur, got %d calls",
			passwordHasher.verifyCall,
		)
	}

	if tokenService.generateCall != 0 {
		t.Fatalf(
			"expected token generation not to occur, got %d calls",
			tokenService.generateCall,
		)
	}
}

// -----------------------------------------------------------------------------
// Token Generation Failure
// -----------------------------------------------------------------------------

func TestAuthenticateUser_TokenGenerationFailure(t *testing.T) {
	testUser := newAuthenticationUser(t)

	repository := &fakeAuthenticationUserRepository{
		user: testUser,
	}

	passwordHasher := &fakeAuthenticationPasswordHasher{}

	tokenGenerationErr := errors.New("token service unavailable")

	tokenService := &fakeAuthenticationTokenService{
		generateErr: tokenGenerationErr,
	}

	useCase := newAuthenticationUseCase(
		t,
		repository,
		passwordHasher,
		tokenService,
	)

	command := dto.LoginUserCommand{
		Email:    testUser.Email().String(),
		Password: "correct-password",
	}

	_, err := useCase.Authenticate(
		context.Background(),
		command,
	)

	if !errors.Is(err, use_cases.ErrTokenGeneration) {
		t.Fatalf(
			"expected ErrTokenGeneration, got %v",
			err,
		)
	}

	if passwordHasher.verifyCall != 1 {
		t.Fatalf(
			"expected password verification to be called once, got %d",
			passwordHasher.verifyCall,
		)
	}

	if tokenService.generateCall != 1 {
		t.Fatalf(
			"expected token generation to be called once, got %d",
			tokenService.generateCall,
		)
	}
}
