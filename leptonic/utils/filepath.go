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

package utils

import (
	"path/filepath"

	"github.com/containerd/nerdctl/v2/leptonic/errs"
)

func RootJoin(root string, args ...string) (string, error) {
	sub := filepath.Join(args...)
	if !filepath.IsAbs(sub) {
		sub = filepath.Join(root, sub)
	}

	remainder := []string{}
	var resolved string

	var err error
	for {
		if resolved, err = filepath.EvalSymlinks(sub); err == nil {
			break
		}
		var rem string
		sub, rem = filepath.Split(sub)
		remainder = append(remainder, rem)
	}

	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return "", err
	}

	rootlen := len(root)
	if len(resolved) < rootlen || resolved[0:rootlen] != root {
		return "", errs.ErrInvalidArgument
	}

	return filepath.Join(append([]string{resolved}, remainder...)...), nil
}
