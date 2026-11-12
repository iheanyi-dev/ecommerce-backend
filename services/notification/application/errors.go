package application

import "errors"

var (
	ErrEmailSenderRequired   = errors.New("email sender is required")
	ErrUnsupportedEmailEvent = errors.New("unsupported email event")
)
