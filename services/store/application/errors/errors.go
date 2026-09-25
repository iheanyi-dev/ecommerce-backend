package errors

import "errors"

var (
	ErrUnauthenticated           = errors.New("unauthenticated")
	ErrInvalidServiceCredentials = errors.New("invalid service credentials")
	ErrStoreAlreadyExists        = errors.New("store already exists")
	ErrStoreSlugAlreadyExists    = errors.New("store slug already exists")
	ErrStoreNotFound             = errors.New("store not found")
	ErrInvalidPagination         = errors.New("invalid pagination")
)
