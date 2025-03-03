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

package helpers

import (
	"errors"
	"strings"

	"github.com/containerd/errdefs"

	"go.farcloser.world/lepton/leptonic/errs"
)

func ErrConvert(err error) error {
	if err == nil {
		return nil
	}

	if errdefs.IsNotFound(err) {
		return errors.Join(errs.ErrNotFound, err)
	}

	if errdefs.IsInvalidArgument(err) {
		return errors.Join(errs.ErrInvalidArgument, err)
	}

	if strings.Contains(err.Error(), "contains value with non-printable ASCII characters") {
		return errors.Join(errs.ErrInvalidArgument, err)
	}

	return errors.Join(errs.ErrSystemFailure, err)
}
