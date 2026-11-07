package use_cases

import "context"

// ResendActivationUseCase handles requests to resend an activation link.
//
// It delegates to the same issuance workflow as the initial request so
// token replacement and account-enumeration protections remain consistent.
type ResendActivationUseCase struct {
	requestActivation *RequestActivationUseCase
}

func NewResendActivationUseCase(
	requestActivation *RequestActivationUseCase,
) *ResendActivationUseCase {
	return &ResendActivationUseCase{
		requestActivation: requestActivation,
	}
}

func (uc *ResendActivationUseCase) Execute(
	ctx context.Context,
	email string,
) error {
	return uc.requestActivation.Execute(ctx, email)
}
