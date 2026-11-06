package errors

import stderrors "errors"

var ErrInvalidPayoutRequest = stderrors.New("invalid payout request")
