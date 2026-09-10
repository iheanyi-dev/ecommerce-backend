package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
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

// UpdateFullName satisfies the UserRepository interface.
//
// Authentication tests do not exercise profile updates, so this fake
// intentionally performs no operation.
func (f *fakeAuthenticationUserRepository) UpdateFullName(
	ctx context.Context,
	existingUser *user.User,
) error {
	return nil
}

func (f *fakeAuthenticationUserRepository) UpdatePasswordHash(
	ctx context.Context,
	existingUser *user.User,
) error {
	return nil
}

// UpdateStatus satisfies the UserRepository interface.
//
// These tests do not exercise account-status persistence.
func (f *fakeAuthenticationUserRepository) UpdateStatus(
	ctx context.Context,
	existingUser *user.User,
) error {
	return nil
}

func (f *fakeAuthenticationUserRepository) List(
	ctx context.Context,
	limit int,
	offset int,
) ([]*user.User, error) {
	return nil, nil
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
// Fake Logger
// -----------------------------------------------------------------------------

// fakeAuthenticationLogger is a test double for the application Logger port.
//
// The logger stores every LogEvent supplied by the use case so tests can
// verify the actual structured event rather than merely checking that
// logging occurred.
type fakeAuthenticationLogger struct {
	events []ports.LogEvent
	logErr error
}

// Log records the supplied structured event.
func (f *fakeAuthenticationLogger) Log(
	ctx context.Context,
	event ports.LogEvent,
) error {
	f.events = append(f.events, event)

	return f.logErr
}

// lastEvent returns the most recently recorded event.
//
// Authentication tests expect at most one authentication event per request.
func (f *fakeAuthenticationLogger) lastEvent(t *testing.T) ports.LogEvent {
	t.Helper()

	if len(f.events) == 0 {
		t.Fatal("expected at least one log event, got none")
	}

	return f.events[len(f.events)-1]
}

// assertAuthenticationEvent verifies the common fields for an authentication
// event.
func assertAuthenticationEvent(
	t *testing.T,
	event ports.LogEvent,
	expectedEvent string,
	expectedUserID string,
	expectedRole string,
	expectedFailureCategory string,
) {
	t.Helper()

	if event.Event != expectedEvent {
		t.Fatalf(
			"expected log event %q, got %q",
			expectedEvent,
			event.Event,
		)
	}

	if event.Operation != "login" {
		t.Fatalf(
			"expected operation %q, got %q",
			"login",
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

	if event.Role != expectedRole {
		t.Fatalf(
			"expected role %q, got %q",
			expectedRole,
			event.Role,
		)
	}

	if event.FailureCategory != expectedFailureCategory {
		t.Fatalf(
			"expected failure category %q, got %q",
			expectedFailureCategory,
			event.FailureCategory,
		)
	}

	// Authentication events must never contain HTTP-specific information
	// unless the presentation layer explicitly supplies it.
	if event.HTTPMethod != "" {
		t.Fatalf(
			"expected HTTP method to be empty, got %q",
			event.HTTPMethod,
		)
	}

	if event.Route != "" {
		t.Fatalf(
			"expected route to be empty, got %q",
			event.Route,
		)
	}

	if event.StatusCode != 0 {
		t.Fatalf(
			"expected status code to be empty, got %d",
			event.StatusCode,
		)
	}

	if event.DurationMillis != 0 {
		t.Fatalf(
			"expected duration to be empty, got %d",
			event.DurationMillis,
		)
	}

	if event.RequiredRole != "" {
		t.Fatalf(
			"expected required role to be empty, got %q",
			event.RequiredRole,
		)
	}
}

// assertAuthenticationEventContainsNoSecrets verifies that authentication
// events do not expose credential material through any structured field.
func assertAuthenticationEventContainsNoSecrets(
	t *testing.T,
	event ports.LogEvent,
) {
	t.Helper()

	// These values intentionally represent secrets that must never appear in
	// authentication log events.
	secrets := []string{
		"correct-password",
		"wrong-password",
		"test-access-token",
		"test-refresh-token",
		"$2a$10$test-password-hash",
	}

	fields := []string{
		event.Event,
		event.Operation,
		event.UserID,
		event.Role,
		event.HTTPMethod,
		event.Route,
		event.FailureCategory,
		event.RequiredRole,
	}

	for _, field := range fields {
		for _, secret := range secrets {
			if field == secret {
				t.Fatalf(
					"authentication log event contains sensitive value %q",
					secret,
				)
			}
		}
	}
}

// -----------------------------------------------------------------------------
// Compile-Time Contract Assertions
// -----------------------------------------------------------------------------

var _ ports.UserRepository = (*fakeAuthenticationUserRepository)(nil)

var _ ports.PasswordHasher = (*fakeAuthenticationPasswordHasher)(nil)

var _ ports.TokenService = (*fakeAuthenticationTokenService)(nil)

var _ ports.Logger = (*fakeAuthenticationLogger)(nil)

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
	logger *fakeAuthenticationLogger,
) *use_cases.AuthenticateUserUseCase {
	t.Helper()

	return use_cases.NewAuthenticateUserUseCase(
		repository,
		passwordHasher,
		tokenService,
		logger,
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

	logger := &fakeAuthenticationLogger{}

	useCase := newAuthenticationUseCase(
		t,
		repository,
		passwordHasher,
		tokenService,
		logger,
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

	if len(logger.events) != 1 {
		t.Fatalf(
			"expected exactly one authentication log event, got %d",
			len(logger.events),
		)
	}

	event := logger.lastEvent(t)

	assertAuthenticationEvent(
		t,
		event,
		"auth.login.succeeded",
		testUser.ID().String(),
		testUser.Role().String(),
		"",
	)

	assertAuthenticationEventContainsNoSecrets(t, event)
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

	logger := &fakeAuthenticationLogger{}

	useCase := newAuthenticationUseCase(
		t,
		repository,
		passwordHasher,
		tokenService,
		logger,
	)

	command := dto.LoginUserCommand{
		Email:    "missing@example.com",
		Password: "correct-password",
	}

	_, err := useCase.Authenticate(
		context.Background(),
		command,
	)

	if !errors.Is(err, application_errors.ErrInvalidCredentials) {
		t.Fatalf(
			"expected ErrInvalidCredentials, got %v",
			err,
		)
	}

	if passwordHasher.verifyCall != 0 {
		t.Fatalf(
			"expected password verification not to be called, got %d calls",
			passwordHasher.verifyCall,
		)
	}

	if tokenService.generateCall != 0 {
		t.Fatalf(
			"expected token generation not to be called, got %d calls",
			tokenService.generateCall,
		)
	}

	if len(logger.events) != 1 {
		t.Fatalf(
			"expected exactly one authentication log event, got %d",
			len(logger.events),
		)
	}

	event := logger.lastEvent(t)

	assertAuthenticationEvent(
		t,
		event,
		"auth.login.user_not_found",
		"",
		"",
		"user_not_found",
	)

	assertAuthenticationEventContainsNoSecrets(t, event)
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

	logger := &fakeAuthenticationLogger{}

	useCase := newAuthenticationUseCase(
		t,
		repository,
		passwordHasher,
		tokenService,
		logger,
	)

	command := dto.LoginUserCommand{
		Email:    testUser.Email().String(),
		Password: "wrong-password",
	}

	_, err := useCase.Authenticate(
		context.Background(),
		command,
	)

	if !errors.Is(err, application_errors.ErrInvalidCredentials) {
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

	if len(logger.events) != 1 {
		t.Fatalf(
			"expected exactly one authentication log event, got %d",
			len(logger.events),
		)
	}

	event := logger.lastEvent(t)

	assertAuthenticationEvent(
		t,
		event,
		"auth.login.invalid_credentials",
		"",
		"",
		"invalid_credentials",
	)

	assertAuthenticationEventContainsNoSecrets(t, event)
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

	logger := &fakeAuthenticationLogger{}

	useCase := newAuthenticationUseCase(
		t,
		repository,
		passwordHasher,
		tokenService,
		logger,
	)

	command := dto.LoginUserCommand{
		Email:    testUser.Email().String(),
		Password: "correct-password",
	}

	_, err := useCase.Authenticate(
		context.Background(),
		command,
	)

	if !errors.Is(err, application_errors.ErrAccountNotActive) {
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

	if len(logger.events) != 1 {
		t.Fatalf(
			"expected exactly one authentication log event, got %d",
			len(logger.events),
		)
	}

	event := logger.lastEvent(t)

	assertAuthenticationEvent(
		t,
		event,
		"auth.login.user_inactive",
		testUser.ID().String(),
		testUser.Role().String(),
		"user_inactive",
	)

	assertAuthenticationEventContainsNoSecrets(t, event)
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

	logger := &fakeAuthenticationLogger{}

	useCase := newAuthenticationUseCase(
		t,
		repository,
		passwordHasher,
		tokenService,
		logger,
	)

	command := dto.LoginUserCommand{
		Email:    pendingUser.Email().String(),
		Password: "correct-password",
	}

	_, err := useCase.Authenticate(
		context.Background(),
		command,
	)

	if !errors.Is(err, application_errors.ErrAccountNotActive) {
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

	if len(logger.events) != 1 {
		t.Fatalf(
			"expected exactly one authentication log event, got %d",
			len(logger.events),
		)
	}

	event := logger.lastEvent(t)

	assertAuthenticationEvent(
		t,
		event,
		"auth.login.user_inactive",
		pendingUser.ID().String(),
		pendingUser.Role().String(),
		"user_inactive",
	)

	assertAuthenticationEventContainsNoSecrets(t, event)
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

	logger := &fakeAuthenticationLogger{}

	useCase := newAuthenticationUseCase(
		t,
		repository,
		passwordHasher,
		tokenService,
		logger,
	)

	command := dto.LoginUserCommand{
		Email:    testUser.Email().String(),
		Password: "correct-password",
	}

	_, err := useCase.Authenticate(
		context.Background(),
		command,
	)

	if !errors.Is(err, application_errors.ErrAccountNotActive) {
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

	if len(logger.events) != 1 {
		t.Fatalf(
			"expected exactly one authentication log event, got %d",
			len(logger.events),
		)
	}

	event := logger.lastEvent(t)

	assertAuthenticationEvent(
		t,
		event,
		"auth.login.user_inactive",
		testUser.ID().String(),
		testUser.Role().String(),
		"user_inactive",
	)

	assertAuthenticationEventContainsNoSecrets(t, event)
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

	logger := &fakeAuthenticationLogger{}

	useCase := newAuthenticationUseCase(
		t,
		repository,
		passwordHasher,
		tokenService,
		logger,
	)

	command := dto.LoginUserCommand{
		Email:    "not-an-email",
		Password: "correct-password",
	}

	_, err := useCase.Authenticate(
		context.Background(),
		command,
	)

	if !errors.Is(err, application_errors.ErrInvalidCredentials) {
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
			"expected token generation not to be called, got %d calls",
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

	logger := &fakeAuthenticationLogger{}

	useCase := newAuthenticationUseCase(
		t,
		repository,
		passwordHasher,
		tokenService,
		logger,
	)

	command := dto.LoginUserCommand{
		Email:    testUser.Email().String(),
		Password: "correct-password",
	}

	_, err := useCase.Authenticate(
		context.Background(),
		command,
	)

	if !errors.Is(err, application_errors.ErrTokenGeneration) {
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

	if len(logger.events) != 1 {
		t.Fatalf(
			"expected exactly one authentication log event, got %d",
			len(logger.events),
		)
	}

	event := logger.lastEvent(t)

	assertAuthenticationEvent(
		t,
		event,
		"auth.login.failed",
		testUser.ID().String(),
		testUser.Role().String(),
		"token_generation_failed",
	)

	assertAuthenticationEventContainsNoSecrets(t, event)
}
