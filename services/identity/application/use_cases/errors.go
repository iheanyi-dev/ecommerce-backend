package use_cases

import "errors"

var (
	// ErrEmailAlreadyExists indicates that a registration attempted to use
	// an email address already associated with another account.
	ErrEmailAlreadyExists = errors.New("email already exists")

	// ErrUserNotFound indicates that the requested user account does not exist.
	ErrUserNotFound = errors.New("user not found")
)
