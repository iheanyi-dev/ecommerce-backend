package errors

import "errors"

var (
	ErrUnauthenticated        = errors.New("unauthenticated")
	ErrStoreAlreadyExists     = errors.New("store already exists")
	ErrStoreSlugAlreadyExists = errors.New("store slug already exists")
	ErrStoreNotFound          = errors.New("store not found")
	ErrInvalidPagination      = errors.New("invalid pagination")
)
