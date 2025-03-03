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
)

func XDGRuntimeDir() (string, error) {
	if xrd := os.Getenv("XDG_RUNTIME_DIR"); xrd != "" {
		return xrd, nil
	}

	// Fall back to "/run/user/<euid>".
	// Note that We cannot rely on os.Geteuid() because we might be inside UserNS.
	euid, err := strconv.Atoi(os.Getenv("ROOTLESSKIT_PARENT_EUID"))
	if err != nil {
		return "", ErrEnvXDGRuntimeDirNotSet
	}

	return "/run/user/" + strconv.Itoa(euid), nil
}

func XDGConfigHome() (string, error) {
	if xch := os.Getenv("XDG_CONFIG_HOME"); xch != "" {
		return xch, nil
	}

	// Fall back to "~/.config"
	// Note that we cannot rely on user.Current().HomeDir because we might be inside UserNS.
	home := os.Getenv("HOME")
	if home == "" {
		return "", ErrEnvHomeNotSet
	}

	return filepath.Join(home, ".config"), nil
}

func XDGDataHome() (string, error) {
	if xdh := os.Getenv("XDG_DATA_HOME"); xdh != "" {
		return xdh, nil
	}

	// Fall back to "~/.local/share"
	// Note that we cannot rely on user.Current().HomeDir because we might be inside UserNS.
	home := os.Getenv("HOME")
	if home == "" {
		return "", ErrEnvHomeNotSet
	}

	return filepath.Join(home, ".local", "share"), nil
}
