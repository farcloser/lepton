package errs

import (
	"errors"
)

var (
	// ErrInvalidArgument describes conditions where an argument value does not match expectations
	// This is usually a user input error, and should be surfaced explicitly as such, along with the (invalid)
	// provided value
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrNotFound typically happens when a certain object that is requested by the user does not exist
	ErrNotFound = errors.New("not found")

	// ErrSystemFailure should be returned by implementations when an internal, unexpected, failure occurs.
	// For example: filesystem errors (typically permissions)
	ErrSystemFailure = errors.New("system failure")

	// ErrFaultyImplementation should be returned by low-level code to make it clear that the way the code is being
	// used is wrong.
	// For example, if a certain abstraction expects to be first initialized before being used
	ErrFaultyImplementation = errors.New("code needs to be fixed")
)
