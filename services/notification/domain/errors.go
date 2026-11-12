package domain

import "errors"

var (
	ErrInvalidEmailAddress = errors.New("invalid email address")
	ErrEmptyEmailSubject   = errors.New("email subject cannot be empty")
	ErrEmptyEmailBody      = errors.New("email body cannot be empty")
	ErrInvalidEmailFormat  = errors.New("invalid email format")
)
