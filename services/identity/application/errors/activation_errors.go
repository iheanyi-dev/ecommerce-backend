package errors

import "errors"

var (
	ErrActivationTokenGeneration = errors.New("activation token generation failed")
	ErrActivationEmailDelivery   = errors.New("activation email delivery failed")
	ErrInvalidActivationURL      = errors.New("invalid activation URL configuration")
)
