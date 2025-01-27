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

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func ChildPid(stateDir string) (int, error) {
	pidFilePath := filepath.Join(stateDir, "child_pid")
	if _, err := os.Stat(pidFilePath); err != nil {
		return 0, err
	}

	pidFileBytes, err := os.ReadFile(pidFilePath)
	if err != nil {
		return 0, err
	}
	pidStr := strings.TrimSpace(string(pidFileBytes))
	return strconv.Atoi(pidStr)
}

func StateDir() (string, error) {
	if v := os.Getenv("ROOTLESSKIT_STATE_DIR"); v != "" {
		return v, nil
	}

	xdr, err := XDGRuntimeDir()
	if err != nil {
		return "", err
	}

	// "${XDG_RUNTIME_DIR}/containerd-rootless" is hardcoded in containerd-rootless.sh
	stateDir := filepath.Join(xdr, "containerd-rootless")
	if _, err := os.Stat(stateDir); err != nil {
		return "", err
	}
	return stateDir, nil
}
