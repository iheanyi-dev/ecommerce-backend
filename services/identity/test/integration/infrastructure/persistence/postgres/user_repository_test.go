package postgres_tests

import (
	"context"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/persistence/postgres"
	generated "github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/persistence/postgres/generated"
	"testing"
	"time"
)

// newTestUser creates a valid domain User for repository tests.
//
// The repository receives domain aggregates, so the tests should construct
// the User through the domain API rather than constructing persistence
// models directly.
func newTestUser(t *testing.T, email string) *user.User {
	t.Helper()

	fullName, err := user.NewFullName("John Doe")
	if err != nil {
		t.Fatalf("failed to create FullName: %v", err)
	}

	userEmail, err := user.NewEmail(email)
	if err != nil {
		t.Fatalf("failed to create Email: %v", err)
	}

	passwordHash, err := user.NewPasswordHash("hashed-password")
	if err != nil {
		t.Fatalf("failed to create PasswordHash: %v", err)
	}

	newUser, err := user.NewUser(
		fullName,
		userEmail,
		passwordHash,
	)
	if err != nil {
		t.Fatalf("failed to create User: %v", err)
	}

	return newUser
}

// TestUserRepository_Create verifies that a valid domain User can be
// persisted through the PostgreSQL repository.
func TestUserRepository_Create(t *testing.T) {
	testDB := NewTestDatabase(t)
	tx := testDB.BeginTx(t)

	// SQLC generates a query executor backed by the transaction.
	queries := generated.New(tx)

	// The repository depends on the SQLC-generated query executor rather
	// than depending directly on PostgreSQL.
	repository := postgres.NewUserRepository(queries)

	newUser := newTestUser(t, "create@example.com")

	err := repository.Create(
		context.Background(),
		newUser,
	)
	if err != nil {
		t.Fatalf("Create() returned an error: %v", err)
	}

	// We verify persistence through the repository's public contract.
	exists, err := repository.ExistsByEmail(
		context.Background(),
		newUser.Email(),
	)
	if err != nil {
		t.Fatalf(
			"ExistsByEmail() returned an error after Create(): %v",
			err,
		)
	}

	if !exists {
		t.Fatal("expected created user to exist")
	}
}

// TestUserRepository_ExistsByEmail_UserDoesNotExist verifies that the
// repository returns false when no user has the requested email.
func TestUserRepository_ExistsByEmail_UserDoesNotExist(t *testing.T) {
	testDB := NewTestDatabase(t)
	tx := testDB.BeginTx(t)

	queries := generated.New(tx)
	repository := postgres.NewUserRepository(queries)

	email, err := user.NewEmail("missing@example.com")
	if err != nil {
		t.Fatalf("failed to create Email: %v", err)
	}

	exists, err := repository.ExistsByEmail(
		context.Background(),
		email,
	)
	if err != nil {
		t.Fatalf("ExistsByEmail() returned an error: %v", err)
	}

	if exists {
		t.Fatal("expected non-existent user to return false")
	}
}

// TestUserRepository_ExistsByEmail_UserExists verifies that the repository
// returns true when a user with the requested email exists.
func TestUserRepository_ExistsByEmail_UserExists(t *testing.T) {
	testDB := NewTestDatabase(t)
	tx := testDB.BeginTx(t)

	queries := generated.New(tx)
	repository := postgres.NewUserRepository(queries)

	newUser := newTestUser(t, "exists@example.com")

	if err := repository.Create(
		context.Background(),
		newUser,
	); err != nil {
		t.Fatalf("Create() returned an error: %v", err)
	}

	exists, err := repository.ExistsByEmail(
		context.Background(),
		newUser.Email(),
	)
	if err != nil {
		t.Fatalf("ExistsByEmail() returned an error: %v", err)
	}

	if !exists {
		t.Fatal("expected existing user to return true")
	}
}

// TestUserRepository_FindByEmail_UserExists verifies that the repository
// retrieves an existing user and correctly reconstructs the domain
// aggregate from the raw values returned by PostgreSQL.
func TestUserRepository_FindByEmail_UserExists(t *testing.T) {
	testDB := NewTestDatabase(t)
	tx := testDB.BeginTx(t)

	queries := generated.New(tx)
	repository := postgres.NewUserRepository(queries)

	newUser := newTestUser(t, "find@example.com")

	if err := repository.Create(
		context.Background(),
		newUser,
	); err != nil {
		t.Fatalf("Create() returned an error: %v", err)
	}

	reconstitutedUser, err := repository.FindByEmail(
		context.Background(),
		newUser.Email(),
	)
	if err != nil {
		t.Fatalf("FindByEmail() returned an error: %v", err)
	}

	if reconstitutedUser == nil {
		t.Fatal("expected user, got nil")
	}

	// Verify that the repository preserved the complete domain identity.
	if reconstitutedUser.ID() != newUser.ID() {
		t.Fatalf("expected ID %s, got %s",
			newUser.ID().String(),
			reconstitutedUser.ID().String(),
		)
	}

	// Verify that the persisted FullName was reconstructed into the
	// domain FullName value object correctly.
	if reconstitutedUser.FullName() != newUser.FullName() {
		t.Fatalf("full name changed during reconstruction")
	}

	// Verify that the persisted Email was reconstructed into the
	// domain Email value object correctly.
	if reconstitutedUser.Email() != newUser.Email() {
		t.Fatalf("email changed during reconstruction")
	}

	// Verify that the persisted PasswordHash was reconstructed without
	// modifying the stored hash.
	if reconstitutedUser.PasswordHash() != newUser.PasswordHash() {
		t.Fatalf("password hash changed during reconstruction")
	}

	// Verify that role and status were reconstructed from their
	// persisted string representations.
	if reconstitutedUser.Role() != newUser.Role() {
		t.Fatalf("role changed during reconstruction")
	}

	if reconstitutedUser.Status() != newUser.Status() {
		t.Fatalf("status changed during reconstruction")
	}

	// PostgreSQL TIMESTAMPTZ stores timestamps with microsecond precision.
	// time.Now() may contain nanoseconds, so compare at the same precision
	// used by PostgreSQL.
	if !reconstitutedUser.CreatedAt().Truncate(time.Microsecond).Equal(
		newUser.CreatedAt().Truncate(time.Microsecond),
	) {
		t.Fatalf("created_at changed during reconstruction")
	}

	if !reconstitutedUser.UpdatedAt().Truncate(time.Microsecond).Equal(
		newUser.UpdatedAt().Truncate(time.Microsecond),
	) {
		t.Fatalf("updated_at changed during reconstruction")
	}
}

// TestUserRepository_FindByEmail_UserDoesNotExist verifies that the
// repository returns the appropriate error when no user exists with
// the requested email.
func TestUserRepository_FindByEmail_UserDoesNotExist(t *testing.T) {
	testDB := NewTestDatabase(t)
	tx := testDB.BeginTx(t)

	queries := generated.New(tx)
	repository := postgres.NewUserRepository(queries)

	email, err := user.NewEmail("missing-find@example.com")
	if err != nil {
		t.Fatalf("failed to create Email: %v", err)
	}

	_, err = repository.FindByEmail(
		context.Background(),
		email,
	)

	if err == nil {
		t.Fatal("expected FindByEmail() to return an error")
	}
}
