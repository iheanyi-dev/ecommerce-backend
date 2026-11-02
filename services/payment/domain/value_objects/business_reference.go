package value_objects

import (
	"strings"

	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/errors"
)

type BusinessReference struct {
	referenceType string
	id            string
}

func NewBusinessReference(referenceType, id string) (BusinessReference, error) {
	referenceType = strings.TrimSpace(referenceType)
	id = strings.TrimSpace(id)

	if referenceType == "" || id == "" {
		return BusinessReference{}, domainerrors.ErrInvalidBusinessReference
	}

	return BusinessReference{
		referenceType: referenceType,
		id:            id,
	}, nil
}

func (r BusinessReference) Type() string {
	return r.referenceType
}

func (r BusinessReference) ID() string {
	return r.id
}

func (r BusinessReference) IsZero() bool {
	return r.referenceType == "" || r.id == ""
}
