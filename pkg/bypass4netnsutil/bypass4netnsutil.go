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

package bypass4netnsutil

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"

	"github.com/containerd/containerd/v2/core/containers"
	"github.com/containerd/containerd/v2/pkg/oci"
	b4nnoci "github.com/rootless-containers/bypass4netns/pkg/oci"

	"go.farcloser.world/containers/specs"
	"go.farcloser.world/core/filesystem"

	"go.farcloser.world/lepton/leptonic/errs"
	"go.farcloser.world/lepton/leptonic/rootlesskit"
	"go.farcloser.world/lepton/pkg/annotations"
)

const (
	b4nnRuntimeDir = "bypass4netns"
	b4nnSocketPath = "bypass4netnsd.sock"
)

func generateSecurityOpt(listenerPath string) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		if s.Linux.Seccomp == nil {
			s.Linux.Seccomp = b4nnoci.GetDefaultSeccompProfile(listenerPath)
		} else {
			sc, err := b4nnoci.TranslateSeccompProfile(*s.Linux.Seccomp, listenerPath)
			if err != nil {
				return errors.Join(errs.ErrInvalidArgument, err)
			}

			s.Linux.Seccomp = sc
		}

		return nil
	}
}

func GenerateBypass4netnsOpts(securityOptsMaps map[string]string, annotationsMap map[string]string, id string) ([]oci.SpecOpts, error) {
	b4nn, ok := annotationsMap[annotations.Bypass4netns]
	if !ok {
		return nil, nil
	}

	b4nnEnable, err := strconv.ParseBool(b4nn)
	if err != nil {
		return nil, errors.Join(errs.ErrInvalidArgument, err)
	}

	if !b4nnEnable {
		return nil, nil
	}

	socketPath, err := getSocketPathByID(id)
	if err != nil {
		return nil, err
	}

	err = ensureRuntimeDir()
	if err != nil {
		return nil, err
	}

	return []oci.SpecOpts{
		generateSecurityOpt(socketPath),
	}, nil
}

func GetBypass4NetnsdDefaultSocketPath() (string, error) {
	xdgRuntimeDir, err := rootlesskit.XDGRuntimeDir()
	if err != nil {
		return "", errors.Join(errs.ErrSystemFailure, err)
	}

	return filepath.Join(xdgRuntimeDir, b4nnSocketPath), nil
}

func ensureRuntimeDir() error {
	runtimeDir, err := getRuntimeDir()
	if err != nil {
		return err
	}

	stat, err := os.Stat(runtimeDir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return errors.Join(errs.ErrSystemFailure, err)
		}

		if err = os.MkdirAll(runtimeDir, filesystem.DirPermissionsPrivate); err != nil {
			return errors.Join(errs.ErrSystemFailure, err)
		}
	} else if !stat.IsDir() {
		return errs.ErrSystemFailure
	}

	return nil
}

func getRuntimeDir() (string, error) {
	xdgRuntimeDir, err := rootlesskit.XDGRuntimeDir()
	if err != nil {
		return "", errors.Join(errs.ErrSystemFailure, err)
	}

	return filepath.Join(xdgRuntimeDir, b4nnRuntimeDir), nil
}

func getSocketPathByID(id string) (string, error) {
	runtimeDir, err := getRuntimeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(runtimeDir, id[0:15]+".sock"), nil
}

func getPidFilePathByID(id string) (string, error) {
	runtimeDir, err := getRuntimeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(runtimeDir, id[0:15]+".pid"), nil
}

func IsBypass4netnsEnabled(annotationsMap map[string]string) (bool, bool, error) {
	b4nn, ok := annotationsMap[annotations.Bypass4netns]
	if !ok {
		return false, false, nil
	}

	enabled, err := strconv.ParseBool(b4nn)
	if err != nil {
		return false, false, errors.Join(errs.ErrInvalidArgument, err)
	}

	bindEnabled := enabled
	if s, ok := annotationsMap[annotations.Bypass4netnsIgnoreBind]; ok {
		var bindDisabled bool
		bindDisabled, err = strconv.ParseBool(s)
		if err != nil {
			return enabled, bindEnabled, errors.Join(errs.ErrInvalidArgument, err)
		}

		bindEnabled = !bindDisabled
	}

	return enabled, bindEnabled, nil
}
