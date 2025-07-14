package errors

import "errors"

var (
	ErrKey          = errors.New("key not exist")
	ErrWrite        = errors.New("error witch write key")
	ErrConflict     = errors.New("conflict url is no exist")
	ErrReedCookie   = errors.New("can't read cookie")
	ErrGenToken     = errors.New("problen with token generation")
	ErrURLIsDeleted = errors.New("url is deleted")
)
var (
	PgUniqueIndexErrorCode = "23505"
)
