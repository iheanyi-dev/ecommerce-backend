package user

import (
	"errors"
	"time"
)

var (
	// ErrUserAlreadyVendor indicates that the user already has vendor privileges.
	ErrUserAlreadyVendor = errors.New("user is already a vendor")

	// ErrInvalidVendorPromotion indicates that the user's current role
	// cannot be promoted to vendor.
	ErrInvalidVendorPromotion = errors.New("user cannot be promoted to vendor")

	// ErrInvalidStatusTransition indicates that the requested account
	// status transition is not allowed by the domain.
	ErrInvalidStatusTransition = errors.New("invalid user status transition")
)

// User is the aggregate root for user identity within the Identity service.
//
// The aggregate owns identity information and account lifecycle state.
// Consumers cannot arbitrarily mutate its internal fields. State changes
// must happen through explicit domain behaviours.
type User struct {
	id           UserID
	fullName     FullName
	email        Email
	passwordHash PasswordHash
	role         Role
	status       Status
	createdAt    time.Time
	updatedAt    time.Time
}

// NewUser creates a new User aggregate.
//
// Every newly registered account begins as a regular user and remains pending
// verification until the appropriate identity workflow activates it.
func NewUser(
	fullName FullName,
	email Email,
	passwordHash PasswordHash,
) (*User, error) {
	now := time.Now().UTC()

	return &User{
		id:           NewUserID(),
		fullName:     fullName,
		email:        email,
		passwordHash: passwordHash,
		role:         RoleUser,
		status:       StatusPendingVerification,
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

// ID returns the user's unique identity.
func (u *User) ID() UserID {
	return u.id
}

// FullName returns the user's current full name.
func (u *User) FullName() FullName {
	return u.fullName
}

// Email returns the user's current email.
func (u *User) Email() Email {
	return u.email
}

// PasswordHash returns the user's current password hash.
//
// The aggregate never exposes the user's raw password.
func (u *User) PasswordHash() PasswordHash {
	return u.passwordHash
}

// Role returns the user's current platform role.
func (u *User) Role() Role {
	return u.role
}

// Status returns the user's current account status.
func (u *User) Status() Status {
	return u.status
}

// CreatedAt returns the time the account was created.
func (u *User) CreatedAt() time.Time {
	return u.createdAt
}

// UpdatedAt returns the time the account was last modified.
func (u *User) UpdatedAt() time.Time {
	return u.updatedAt
}

// ChangeFullName replaces the user's full name with another validated
// FullName value and records the modification time.
func (u *User) ChangeFullName(fullName FullName) {
	u.fullName = fullName
	u.touch()
}

// ChangeEmail replaces the user's email with another validated Email value.
//
// Email validation itself belongs to the Email value object. The aggregate
// simply replaces the old immutable value with the new immutable value.
func (u *User) ChangeEmail(email Email) {
	u.email = email
	u.touch()
}

// ChangePassword replaces the user's password hash.
//
// Password hashing is performed outside the domain. The aggregate receives
// only the resulting validated PasswordHash value.
func (u *User) ChangePassword(passwordHash PasswordHash) {
	u.passwordHash = passwordHash
	u.touch()
}

// PromoteToVendor changes a regular user into a vendor.
//
// Becoming a vendor means the user has chosen to participate as a seller
// within the marketplace. Creating the actual Store remains the responsibility
// of the Store domain/service.
//
// This operation can only perform USER -> VENDOR. It cannot be used to grant
// administrative privileges.
func (u *User) PromoteToVendor() error {
	if u.role == RoleVendor {
		return ErrUserAlreadyVendor
	}

	if u.role != RoleUser {
		return ErrInvalidVendorPromotion
	}

	u.role = RoleVendor
	u.touch()

	return nil
}

// Activate moves a user into the active account state.
//
// Both pending-verification and inactive accounts can become active through
// the appropriate application workflow. Suspended accounts require a
// separate domain decision and cannot be activated blindly.
func (u *User) Activate() error {
	switch u.status {
	case StatusPendingVerification, StatusInactive:
		u.status = StatusActive
		u.touch()
		return nil

	default:
		return ErrInvalidStatusTransition
	}
}

// Suspend places an active account into the suspended state.
func (u *User) Suspend() error {
	if u.status != StatusActive {
		return ErrInvalidStatusTransition
	}

	u.status = StatusSuspended
	u.touch()

	return nil
}

// Deactivate places an active account into the inactive state.
func (u *User) Deactivate() error {
	if u.status != StatusActive {
		return ErrInvalidStatusTransition
	}

	u.status = StatusInactive
	u.touch()

	return nil
}

// touch updates the aggregate's modification timestamp.
//
// Keeping this operation private prevents callers from manipulating
// UpdatedAt independently of an actual domain state change.
func (u *User) touch() {
	u.updatedAt = time.Now().UTC()
}
