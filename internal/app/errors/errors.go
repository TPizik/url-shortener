package errors

import "errors"

var ErrKey = errors.New("key not exist")
var ErrWrite = errors.New("error witch write key")
var ErrConflict = errors.New("conflict url is no exist")
var ErrReedCookie = errors.New("can't read cookie")
var ErrGenToken = errors.New("problen with token generation")
