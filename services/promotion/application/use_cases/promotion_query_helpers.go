package use_cases

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/dtos"
	apperrors "github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/entities"
)

func validatePagination(limit, offset int) error {
	if limit <= 0 || offset < 0 {
		return apperrors.ErrInvalidPromotionQuery
	}
	return nil
}

func validateUUID(name, value string) error {
	if _, err := uuid.Parse(value); err != nil {
		return fmt.Errorf("%w: invalid %s", apperrors.ErrInvalidPromotionQuery, name)
	}
	return nil
}

func toPromotionDTOs(promotions []*entities.Promotion, now time.Time) []dtos.PromotionDTO {
	result := make([]dtos.PromotionDTO, 0, len(promotions))
	for _, promotion := range promotions {
		if promotion == nil {
			continue
		}
		result = append(result, toPromotionDTO(promotion, now))
	}
	return result
}
