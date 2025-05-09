/*
   Copyright Farcloser.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

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

	ErrCancelled = errors.New("operation cancelled")

	// ErrNetworkCondition is meant to wrap network level errors - DNS, TCP, TLS errors
	// but NOT http server level errors
	ErrNetworkCondition = errors.New("network communication failed")

	// ErrServerIsMisbehaving should wrap all server errors (eg: status code 50x)
	// but NOT dns, tcp, or tls errors
	ErrServerIsMisbehaving = errors.New("server error")

	// ErrFailedPrecondition should wrap errors encountered when a certain operation cannot be performed because
	// a precondition prevents it from completing. For example, removing a volume that is in use.
	ErrFailedPrecondition = errors.New("unable to perform the requested operation")
)
