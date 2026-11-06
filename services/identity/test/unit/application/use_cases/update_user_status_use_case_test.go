package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	domain_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

type fakeUpdateStatusRepository struct {
	user             *user.User
	findByIDErr      error
	updateStatusErr  error
	updateStatusCall bool
	updatedUser      *user.User
}

func (f *fakeUpdateStatusRepository) ExistsByEmail(context.Context, user.Email) (bool, error) {
	return false, nil
}

func (f *fakeUpdateStatusRepository) Create(context.Context, *user.User) error {
	return nil
}

func (f *fakeUpdateStatusRepository) FindByEmail(context.Context, user.Email) (*user.User, error) {
	return nil, nil
}

func (f *fakeUpdateStatusRepository) FindByID(context.Context, user.UserID) (*user.User, error) {
	if f.findByIDErr != nil {
		return nil, f.findByIDErr
	}
	return f.user, nil
}

func (f *fakeUpdateStatusRepository) UpdateFullName(context.Context, *user.User) error {
	return nil
}

func (f *fakeUpdateStatusRepository) UpdatePasswordHash(context.Context, *user.User) error {
	return nil
}

func (f *fakeUpdateStatusRepository) UpdateRole(context.Context, *user.User) error {
	return nil
}

func (f *fakeUpdateStatusRepository) UpdateStatus(_ context.Context, existingUser *user.User) error {
	f.updateStatusCall = true
	f.updatedUser = existingUser
	return f.updateStatusErr
}

func (f *fakeUpdateStatusRepository) List(context.Context, int, int) ([]*user.User, error) {
	return nil, nil
}

type fakeUpdateStatusLogger struct{}

func (f *fakeUpdateStatusLogger) Log(context.Context, ports.LogEvent) error {
	return nil
}

func newUpdateStatusUser(t *testing.T, active bool) *user.User {
	t.Helper()

	fullName, err := user.NewFullName("Test User")
	if err != nil {
		t.Fatal(err)
	}
	email, err := user.NewEmail("test@example.com")
	if err != nil {
		t.Fatal(err)
	}
	passwordHash, err := user.NewPasswordHash("hashed-password")
	if err != nil {
		t.Fatal(err)
	}

	newUser, err := user.NewUser(fullName, email, passwordHash)
	if err != nil {
		t.Fatal(err)
	}

	if active {
		if err := newUser.VerifyEmail(); err != nil {
			t.Fatal(err)
		}
	}

	return newUser
}

func newUpdateStatusUseCase(repository *fakeUpdateStatusRepository) *use_cases.UpdateUserStatusUseCase {
	return use_cases.NewUpdateUserStatusUseCase(repository)
}

func TestUpdateUserStatusUseCase_SuspendsActiveUser(t *testing.T) {
	existingUser := newUpdateStatusUser(t, true)
	repository := &fakeUpdateStatusRepository{user: existingUser}

	result, err := newUpdateStatusUseCase(repository).Execute(
		context.Background(),
		existingUser.ID().String(),
		dto.UpdateUserStatusCommand{Status: string(user.StatusSuspended)},
	)
	if err != nil {
		t.Fatalf("expected suspension to succeed, got %v", err)
	}
	if result.Status != string(user.StatusSuspended) {
		t.Fatalf("expected suspended status, got %q", result.Status)
	}
	if !repository.updateStatusCall || repository.updatedUser != existingUser {
		t.Fatal("expected updated aggregate to be persisted")
	}
}

func TestUpdateUserStatusUseCase_ReactivatesSuspendedUser(t *testing.T) {
	existingUser := newUpdateStatusUser(t, true)
	if err := existingUser.Suspend(); err != nil {
		t.Fatal(err)
	}
	repository := &fakeUpdateStatusRepository{user: existingUser}

	result, err := newUpdateStatusUseCase(repository).Execute(
		context.Background(),
		existingUser.ID().String(),
		dto.UpdateUserStatusCommand{Status: string(user.StatusActive)},
	)
	if err != nil {
		t.Fatalf("expected reactivation to succeed, got %v", err)
	}
	if result.Status != string(user.StatusActive) {
		t.Fatalf("expected active status, got %q", result.Status)
	}
	if !repository.updateStatusCall {
		t.Fatal("expected UpdateStatus to be called")
	}
}

func TestUpdateUserStatusUseCase_CannotActivatePendingUser(t *testing.T) {
	existingUser := newUpdateStatusUser(t, false)
	repository := &fakeUpdateStatusRepository{user: existingUser}

	_, err := newUpdateStatusUseCase(repository).Execute(
		context.Background(),
		existingUser.ID().String(),
		dto.UpdateUserStatusCommand{Status: string(user.StatusActive)},
	)
	if !errors.Is(err, domain_errors.ErrInvalidStatusTransition) {
		t.Fatalf("expected invalid transition error, got %v", err)
	}
	if repository.updateStatusCall {
		t.Fatal("must not persist an invalid transition")
	}
}

func TestUpdateUserStatusUseCase_ReturnsNotFound(t *testing.T) {
	repository := &fakeUpdateStatusRepository{}

	_, err := newUpdateStatusUseCase(repository).Execute(
		context.Background(),
		uuid.NewString(),
		dto.UpdateUserStatusCommand{Status: string(user.StatusSuspended)},
	)
	if !errors.Is(err, application_errors.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
	if repository.updateStatusCall {
		t.Fatal("must not persist when user is missing")
	}
}

func TestUpdateUserStatusUseCase_RejectsInvalidUserID(t *testing.T) {
	repository := &fakeUpdateStatusRepository{}

	_, err := newUpdateStatusUseCase(repository).Execute(
		context.Background(),
		"invalid-id",
		dto.UpdateUserStatusCommand{Status: string(user.StatusSuspended)},
	)
	if err == nil {
		t.Fatal("expected invalid user ID error")
	}
	if repository.updateStatusCall {
		t.Fatal("must not persist an invalid request")
	}
}

func TestUpdateUserStatusUseCase_PropagatesLookupError(t *testing.T) {
	expectedErr := errors.New("lookup failed")
	repository := &fakeUpdateStatusRepository{findByIDErr: expectedErr}

	_, err := newUpdateStatusUseCase(repository).Execute(
		context.Background(),
		uuid.NewString(),
		dto.UpdateUserStatusCommand{Status: string(user.StatusSuspended)},
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected lookup error, got %v", err)
	}
}

func TestUpdateUserStatusUseCase_PropagatesPersistenceError(t *testing.T) {
	existingUser := newUpdateStatusUser(t, true)
	expectedErr := errors.New("update failed")
	repository := &fakeUpdateStatusRepository{
		user:            existingUser,
		updateStatusErr: expectedErr,
	}

	_, err := newUpdateStatusUseCase(repository).Execute(
		context.Background(),
		existingUser.ID().String(),
		dto.UpdateUserStatusCommand{Status: string(user.StatusSuspended)},
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected persistence error, got %v", err)
	}
	if !repository.updateStatusCall {
		t.Fatal("expected UpdateStatus to be called")
	}
}

func TestUpdateUserStatusUseCase_RejectsUnsupportedStatus(t *testing.T) {
	existingUser := newUpdateStatusUser(t, true)
	repository := &fakeUpdateStatusRepository{user: existingUser}

	_, err := newUpdateStatusUseCase(repository).Execute(
		context.Background(),
		existingUser.ID().String(),
		dto.UpdateUserStatusCommand{Status: "deleted"},
	)
	if !errors.Is(err, domain_errors.ErrInvalidStatusTransition) {
		t.Fatalf("expected invalid status transition, got %v", err)
	}
	if repository.updateStatusCall {
		t.Fatal("must not persist unsupported status")
	}
}

var _ ports.UserRepository = (*fakeUpdateStatusRepository)(nil)
var _ ports.Logger = (*fakeUpdateStatusLogger)(nil)
