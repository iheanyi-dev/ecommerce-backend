package use_cases

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	domain_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

type fakePromoteVendorRepository struct {
	user             *user.User
	findByIDErr      error
	updateRoleErr    error
	updateRoleCalled bool
	updatedUser      *user.User
}

func (f *fakePromoteVendorRepository) ExistsByEmail(
	context.Context,
	user.Email,
) (bool, error) {
	return false, nil
}

func (f *fakePromoteVendorRepository) Create(
	context.Context,
	*user.User,
) error {
	return nil
}

func (f *fakePromoteVendorRepository) FindByEmail(
	context.Context,
	user.Email,
) (*user.User, error) {
	return nil, nil
}

func (f *fakePromoteVendorRepository) FindByID(
	context.Context,
	user.UserID,
) (*user.User, error) {
	if f.findByIDErr != nil {
		return nil, f.findByIDErr
	}

	return f.user, nil
}

func (f *fakePromoteVendorRepository) UpdateFullName(
	context.Context,
	*user.User,
) error {
	return nil
}

func (f *fakePromoteVendorRepository) UpdatePasswordHash(
	context.Context,
	*user.User,
) error {
	return nil
}

func (f *fakePromoteVendorRepository) UpdateRole(
	_ context.Context,
	existingUser *user.User,
) error {
	f.updateRoleCalled = true
	f.updatedUser = existingUser
	return f.updateRoleErr
}

func (f *fakePromoteVendorRepository) UpdateStatus(
	context.Context,
	*user.User,
) error {
	return nil
}

func (f *fakePromoteVendorRepository) List(
	context.Context,
	int,
	int,
) ([]*user.User, error) {
	return nil, nil
}

type fakePromoteVendorLogger struct{}

func (f *fakePromoteVendorLogger) Log(
	context.Context,
	ports.LogEvent,
) error {
	return nil
}

func newPromoteVendorUser(t *testing.T) *user.User {
	t.Helper()

	fullName, err := user.NewFullName("Test User")
	if err != nil {
		t.Fatalf("failed to create full name: %v", err)
	}

	email, err := user.NewEmail("test@example.com")
	if err != nil {
		t.Fatalf("failed to create email: %v", err)
	}

	passwordHash, err := user.NewPasswordHash("hashed-password")
	if err != nil {
		t.Fatalf("failed to create password hash: %v", err)
	}

	newUser, err := user.NewUser(
		fullName,
		email,
		passwordHash,
	)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	if err := newUser.Activate(); err != nil {
		t.Fatalf("failed to activate user: %v", err)
	}

	return newUser
}

func TestPromoteUserToVendorUseCase_PromotesUser(t *testing.T) {
	existingUser := newPromoteVendorUser(t)

	repository := &fakePromoteVendorRepository{
		user: existingUser,
	}

	useCase := use_cases.NewPromoteUserToVendorUseCase(
		repository,
		&fakePromoteVendorLogger{},
	)

	err := useCase.Execute(
		context.Background(),
		existingUser.ID().String(),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !repository.updateRoleCalled {
		t.Fatal("expected UpdateRole to be called")
	}

	if repository.updatedUser != existingUser {
		t.Fatal("expected the existing user aggregate to be persisted")
	}

	if existingUser.Role() != user.RoleVendor {
		t.Fatalf("expected vendor role, got %s", existingUser.Role())
	}
}

func TestPromoteUserToVendorUseCase_ReturnsNotFound(t *testing.T) {
	repository := &fakePromoteVendorRepository{}

	useCase := use_cases.NewPromoteUserToVendorUseCase(
		repository,
		&fakePromoteVendorLogger{},
	)

	err := useCase.Execute(
		context.Background(),
		uuid.New().String(),
	)
	if !errors.Is(err, application_errors.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}

	if repository.updateRoleCalled {
		t.Fatal("expected UpdateRole not to be called")
	}
}

func TestPromoteUserToVendorUseCase_RejectsAlreadyVendor(t *testing.T) {
	existingUser := newPromoteVendorUser(t)

	if err := existingUser.PromoteToVendor(); err != nil {
		t.Fatalf("failed to promote test user: %v", err)
	}

	repository := &fakePromoteVendorRepository{
		user: existingUser,
	}

	useCase := use_cases.NewPromoteUserToVendorUseCase(
		repository,
		&fakePromoteVendorLogger{},
	)

	err := useCase.Execute(
		context.Background(),
		existingUser.ID().String(),
	)
	if !errors.Is(err, domain_errors.ErrUserAlreadyVendor) {
		t.Fatalf("expected ErrUserAlreadyVendor, got %v", err)
	}

	if repository.updateRoleCalled {
		t.Fatal("expected UpdateRole not to be called")
	}
}

func TestPromoteUserToVendorUseCase_PropagatesRepositoryLookupError(t *testing.T) {
	expectedErr := errors.New("find user failed")

	repository := &fakePromoteVendorRepository{
		findByIDErr: expectedErr,
	}

	useCase := use_cases.NewPromoteUserToVendorUseCase(
		repository,
		&fakePromoteVendorLogger{},
	)

	err := useCase.Execute(
		context.Background(),
		uuid.New().String(),
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
}

func TestPromoteUserToVendorUseCase_PropagatesUpdateRoleError(t *testing.T) {
	existingUser := newPromoteVendorUser(t)
	expectedErr := errors.New("update role failed")

	repository := &fakePromoteVendorRepository{
		user:          existingUser,
		updateRoleErr: expectedErr,
	}

	useCase := use_cases.NewPromoteUserToVendorUseCase(
		repository,
		&fakePromoteVendorLogger{},
	)

	err := useCase.Execute(
		context.Background(),
		existingUser.ID().String(),
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected UpdateRole error, got %v", err)
	}

	if !repository.updateRoleCalled {
		t.Fatal("expected UpdateRole to be called")
	}
}

func TestPromoteUserToVendorUseCase_RejectsInvalidUserID(t *testing.T) {
	repository := &fakePromoteVendorRepository{}

	useCase := use_cases.NewPromoteUserToVendorUseCase(
		repository,
		&fakePromoteVendorLogger{},
	)

	err := useCase.Execute(
		context.Background(),
		"not-a-uuid",
	)
	if err == nil {
		t.Fatal("expected invalid user ID error")
	}

	if repository.updateRoleCalled {
		t.Fatal("expected UpdateRole not to be called")
	}
}

var _ ports.UserRepository = (*fakePromoteVendorRepository)(nil)
var _ ports.Logger = (*fakePromoteVendorLogger)(nil)
