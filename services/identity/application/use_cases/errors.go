package use_cases

import "errors"

var (
	// ErrEmailAlreadyExists indicates that another user already owns
	// the requested email address.
	ErrEmailAlreadyExists = errors.New("email already exists")
)
