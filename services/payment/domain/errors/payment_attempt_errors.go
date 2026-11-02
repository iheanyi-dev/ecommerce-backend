package errors

import stderrors "errors"

var (
	ErrInvalidPaymentAttempt           = stderrors.New("invalid payment attempt")
	ErrInvalidPaymentAttemptTransition = stderrors.New("invalid payment attempt state transition")
)
