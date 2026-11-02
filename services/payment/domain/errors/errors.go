package errors

import stderrors "errors"

var (
	ErrInvalidPayment           = stderrors.New("invalid payment")
	ErrInvalidPaymentTransition = stderrors.New("invalid payment state transition")
	ErrInvalidMoney             = stderrors.New("invalid money")
	ErrInvalidBusinessReference = stderrors.New("invalid business reference")
)
