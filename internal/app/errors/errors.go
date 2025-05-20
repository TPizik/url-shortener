package errors

import "errors"

var ErrKey error = errors.New("key not exist")
var ErrWrite error = errors.New("error witch write key")
