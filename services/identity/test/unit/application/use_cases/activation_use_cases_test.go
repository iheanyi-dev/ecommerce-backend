package use_cases_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/activation"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

type activationUserRepository struct {
	existingUser *user.User
	findErr      error
}

func (f *activationUserRepository) ExistsByEmail(context.Context, user.Email) (bool, error) {
	return f.existingUser != nil, nil
}
func (f *activationUserRepository) Create(context.Context, *user.User) error {
	return nil
}
func (f *activationUserRepository) FindByEmail(context.Context, user.Email) (*user.User, error) {
	return f.existingUser, f.findErr
}
func (f *activationUserRepository) FindByID(context.Context, user.UserID) (*user.User, error) {
	return nil, nil
}
func (f *activationUserRepository) UpdateFullName(context.Context, *user.User) error {
	return nil
}
func (f *activationUserRepository) UpdatePasswordHash(context.Context, *user.User) error {
	return nil
}
func (f *activationUserRepository) UpdateRole(context.Context, *user.User) error {
	return nil
}
func (f *activationUserRepository) UpdateStatus(context.Context, *user.User) error {
	return nil
}
func (f *activationUserRepository) List(context.Context, int, int) ([]*user.User, error) {
	return nil, nil
}

type activationTokenRepository struct {
	createdToken *activation.Token
	deletedUser  user.UserID
	deleteCalls  int
	consumeHash  string
	consumeAt    time.Time
	createErr    error
	deleteErr    error
	consumeErr   error
}

func (f *activationTokenRepository) Create(_ context.Context, token *activation.Token) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.createdToken = token
	return nil
}
func (f *activationTokenRepository) DeleteByUserID(_ context.Context, id user.UserID) error {
	f.deleteCalls++
	f.deletedUser = id
	return f.deleteErr
}
func (f *activationTokenRepository) ConsumeAndActivate(
	_ context.Context,
	hash string,
	at time.Time,
) error {
	f.consumeHash = hash
	f.consumeAt = at
	return f.consumeErr
}

type activationEmailSender struct {
	email user.Email
	link  string
	err   error
	calls int
}

func (f *activationEmailSender) SendActivationEmail(
	_ context.Context,
	email user.Email,
	link string,
) error {
	f.calls++
	f.email = email
	f.link = link
	return f.err
}

func newActivationTestUser(t *testing.T) *user.User {
	t.Helper()

	name, err := user.NewFullName("Activation Test")
	if err != nil {
		t.Fatal(err)
	}
	email, err := user.NewEmail("activation@example.com")
	if err != nil {
		t.Fatal(err)
	}
	password, err := user.NewPasswordHash("hashed-password")
	if err != nil {
		t.Fatal(err)
	}

	existingUser, err := user.NewUser(name, email, password)
	if err != nil {
		t.Fatal(err)
	}
	return existingUser
}

func newActivationRequest(
	repository *activationTokenRepository,
	sender *activationEmailSender,
	existingUser *user.User,
) *use_cases.RequestActivationUseCase {
	return use_cases.NewRequestActivationUseCase(
		&activationUserRepository{existingUser: existingUser},
		repository,
		sender,
		"https://example.com/activate",
		time.Hour,
	)
}

func TestRequestActivationUseCase_IssuesAndEmailsToken(t *testing.T) {
	existingUser := newActivationTestUser(t)
	repository := &activationTokenRepository{}
	sender := &activationEmailSender{}
	uc := newActivationRequest(repository, sender, existingUser)

	if err := uc.Execute(context.Background(), "activation@example.com"); err != nil {
		t.Fatalf("expected activation request to succeed, got %v", err)
	}

	if repository.createdToken == nil {
		t.Fatal("expected activation token to be persisted")
	}
	if repository.createdToken.UserID() != existingUser.ID() {
		t.Fatal("expected token to belong to the requested user")
	}
	if repository.deleteCalls != 1 {
		t.Fatalf("expected previous tokens to be invalidated once, got %d", repository.deleteCalls)
	}
	if sender.calls != 1 {
		t.Fatalf("expected one activation email, got %d", sender.calls)
	}
	if sender.email != existingUser.Email() {
		t.Fatal("expected activation email to target the user's email")
	}

	parsed, err := url.Parse(sender.link)
	if err != nil {
		t.Fatalf("expected valid activation URL, got %v", err)
	}
	rawToken := parsed.Query().Get("token")
	if rawToken == "" {
		t.Fatal("expected activation URL to contain a token")
	}

	hash := sha256.Sum256([]byte(rawToken))
	expectedHash := hex.EncodeToString(hash[:])
	if repository.createdToken.TokenHash() != expectedHash {
		t.Fatal("expected only the SHA-256 token hash to be persisted")
	}
	if strings.Contains(repository.createdToken.TokenHash(), rawToken) {
		t.Fatal("raw token must not be persisted")
	}
}

func TestRequestActivationUseCase_DoesNotEmailUnknownUser(t *testing.T) {
	repository := &activationTokenRepository{}
	sender := &activationEmailSender{}
	uc := newActivationRequest(repository, sender, nil)

	if err := uc.Execute(context.Background(), "unknown@example.com"); err != nil {
		t.Fatalf("expected generic success for unknown email, got %v", err)
	}
	if sender.calls != 0 || repository.createdToken != nil {
		t.Fatal("must not issue a token or send email for an unknown user")
	}
}

func TestRequestActivationUseCase_DoesNotEmailActiveUser(t *testing.T) {
	existingUser := newActivationTestUser(t)
	if err := existingUser.VerifyEmail(); err != nil {
		t.Fatal(err)
	}

	repository := &activationTokenRepository{}
	sender := &activationEmailSender{}
	uc := newActivationRequest(repository, sender, existingUser)

	if err := uc.Execute(context.Background(), "activation@example.com"); err != nil {
		t.Fatalf("expected generic success for active user, got %v", err)
	}
	if sender.calls != 0 || repository.createdToken != nil {
		t.Fatal("must not issue a token or send email for an active user")
	}
}

func TestRequestActivationUseCase_PropagatesLookupFailure(t *testing.T) {
	expectedErr := errors.New("lookup failed")
	repository := &activationUserRepository{findErr: expectedErr}
	uc := use_cases.NewRequestActivationUseCase(
		repository,
		&activationTokenRepository{},
		&activationEmailSender{},
		"https://example.com/activate",
		time.Hour,
	)

	err := uc.Execute(context.Background(), "activation@example.com")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected lookup error, got %v", err)
	}
}

func TestRequestActivationUseCase_CleansUpTokenWhenEmailFails(t *testing.T) {
	existingUser := newActivationTestUser(t)
	repository := &activationTokenRepository{}
	sender := &activationEmailSender{err: errors.New("SMTP unavailable")}
	uc := newActivationRequest(repository, sender, existingUser)

	err := uc.Execute(context.Background(), "activation@example.com")
	if err == nil {
		t.Fatal("expected email delivery failure")
	}
	if repository.createdToken == nil {
		t.Fatal("expected token creation before email delivery")
	}
	if repository.deleteCalls != 2 {
		t.Fatalf("expected old and newly-issued tokens to be deleted, got %d calls", repository.deleteCalls)
	}
}

func TestRequestActivationUseCase_RejectsInvalidActivationURL(t *testing.T) {
	existingUser := newActivationTestUser(t)
	repository := &activationTokenRepository{}
	sender := &activationEmailSender{}
	uc := use_cases.NewRequestActivationUseCase(
		&activationUserRepository{existingUser: existingUser},
		repository,
		sender,
		"://invalid",
		time.Hour,
	)

	if err := uc.Execute(context.Background(), "activation@example.com"); err == nil {
		t.Fatal("expected invalid activation URL error")
	}
	if repository.createdToken != nil || sender.calls != 0 {
		t.Fatal("must not create a token or send email with an invalid URL")
	}
}

func TestResendActivationUseCase_DelegatesToRequestWorkflow(t *testing.T) {
	existingUser := newActivationTestUser(t)
	repository := &activationTokenRepository{}
	sender := &activationEmailSender{}
	request := newActivationRequest(repository, sender, existingUser)
	resend := use_cases.NewResendActivationUseCase(request)

	if err := resend.Execute(context.Background(), "activation@example.com"); err != nil {
		t.Fatalf("expected resend to succeed, got %v", err)
	}
	if sender.calls != 1 || repository.createdToken == nil {
		t.Fatal("expected resend to issue and email a fresh token")
	}
}

func TestVerifyEmailUseCase_HashesTokenAndConsumesIt(t *testing.T) {
	repository := &activationTokenRepository{}
	uc := use_cases.NewVerifyEmailUseCase(repository)
	rawToken := "a-valid-activation-token"

	if err := uc.Execute(context.Background(), rawToken); err != nil {
		t.Fatalf("expected verification to succeed, got %v", err)
	}

	hash := sha256.Sum256([]byte(rawToken))
	if repository.consumeHash != hex.EncodeToString(hash[:]) {
		t.Fatal("expected verification to pass the token hash to persistence")
	}
	if repository.consumeAt.IsZero() {
		t.Fatal("expected verification timestamp to be supplied")
	}
}

func TestVerifyEmailUseCase_RejectsEmptyToken(t *testing.T) {
	repository := &activationTokenRepository{}
	uc := use_cases.NewVerifyEmailUseCase(repository)

	if err := uc.Execute(context.Background(), "  "); err == nil {
		t.Fatal("expected empty token to be rejected")
	}
	if repository.consumeHash != "" {
		t.Fatal("must not call persistence for an empty token")
	}
}

func TestVerifyEmailUseCase_PropagatesPersistenceFailure(t *testing.T) {
	expectedErr := errors.New("invalid or expired token")
	repository := &activationTokenRepository{consumeErr: expectedErr}
	uc := use_cases.NewVerifyEmailUseCase(repository)

	err := uc.Execute(context.Background(), "some-token")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected persistence error, got %v", err)
	}
}

var _ ports.UserRepository = (*activationUserRepository)(nil)
var _ ports.ActivationTokenRepository = (*activationTokenRepository)(nil)
var _ ports.ActivationEmailSender = (*activationEmailSender)(nil)
