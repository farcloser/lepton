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

package rootlesskit

import "errors"

var (
	ErrEnvXDGRuntimeDirNotSet = errors.New("environment variable XDG_RUNTIME_DIR is not set, see https://rootlesscontaine.rs/getting-started/common/login/")
	ErrEnvHomeNotSet          = errors.New("environment variable HOME is not set")
	ErrXDGNotAvailable        = errors.New("can only query XDG env vars on Linux")
)
