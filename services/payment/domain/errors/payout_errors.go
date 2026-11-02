package errors

import stderrors "errors"

var (
	ErrInvalidPayout                  = stderrors.New("invalid store payout")
	ErrInvalidPayoutTransition        = stderrors.New("invalid store payout state transition")
	ErrInvalidPayoutAttempt           = stderrors.New("invalid store payout attempt")
	ErrInvalidPayoutAttemptTransition = stderrors.New("invalid store payout attempt state transition")
)
