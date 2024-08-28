package errs

import (
	"errors"
)

var (
	ErrInvalidArgument    = errors.New("invalid argument provided")
	ErrNotFound           = errors.New("requested resource could not be found")
	ErrSystemFailure      = errors.New("system failure")
	ErrFailedPrecondition = errors.New("failed precondition")
)
