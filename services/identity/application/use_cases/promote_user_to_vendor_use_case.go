package use_cases

import (
	"context"

	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

type PromoteUserToVendorUseCase struct {
	userRepository ports.UserRepository
	logger         ports.Logger
}

func NewPromoteUserToVendorUseCase(
	userRepository ports.UserRepository,
	logger ports.Logger,
) *PromoteUserToVendorUseCase {
	return &PromoteUserToVendorUseCase{
		userRepository: userRepository,
		logger:         logger,
	}
}

func (uc *PromoteUserToVendorUseCase) Execute(
	ctx context.Context,
	userID string,
) error {
	id, err := user.UserIDFromString(userID)
	if err != nil {
		uc.logPromotionEvent(ctx, ports.LogEvent{
			Event:           "user.vendor_promotion.validation_failed",
			Operation:       "promote_user_to_vendor",
			FailureCategory: "validation_failed",
		})
		return err
	}

	existingUser, err := uc.userRepository.FindByID(ctx, id)
	if err != nil {
		uc.logPromotionEvent(ctx, ports.LogEvent{
			Event:           "user.vendor_promotion.failed",
			Operation:       "promote_user_to_vendor",
			UserID:          id.String(),
			FailureCategory: "user_lookup_failed",
		})
		return err
	}

	if existingUser == nil {
		uc.logPromotionEvent(ctx, ports.LogEvent{
			Event:           "user.vendor_promotion.not_found",
			Operation:       "promote_user_to_vendor",
			UserID:          id.String(),
			FailureCategory: "user_not_found",
		})
		return application_errors.ErrUserNotFound
	}

	if err := existingUser.PromoteToVendor(); err != nil {
		uc.logPromotionEvent(ctx, ports.LogEvent{
			Event:           "user.vendor_promotion.validation_failed",
			Operation:       "promote_user_to_vendor",
			UserID:          existingUser.ID().String(),
			Role:            string(existingUser.Role()),
			FailureCategory: "validation_failed",
		})
		return err
	}

	if err := uc.userRepository.UpdateRole(ctx, existingUser); err != nil {
		uc.logPromotionEvent(ctx, ports.LogEvent{
			Event:           "user.vendor_promotion.failed",
			Operation:       "promote_user_to_vendor",
			UserID:          existingUser.ID().String(),
			Role:            string(existingUser.Role()),
			FailureCategory: "role_update_failed",
		})
		return err
	}

	uc.logPromotionEvent(ctx, ports.LogEvent{
		Event:     "user.vendor_promotion.succeeded",
		Operation: "promote_user_to_vendor",
		UserID:    existingUser.ID().String(),
		Role:      string(existingUser.Role()),
	})

	return nil
}

func (uc *PromoteUserToVendorUseCase) logPromotionEvent(
	ctx context.Context,
	event ports.LogEvent,
) {
	if uc.logger == nil {
		return
	}

	_ = uc.logger.Log(ctx, event)
}

var _ ports.PromoteUserToVendorService = (*PromoteUserToVendorUseCase)(nil)
