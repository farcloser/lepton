//go:build !linux

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

package ocihook

import (
	"context"
	"errors"

	"github.com/containerd/go-cni"
	rlkclient "github.com/rootless-containers/rootlesskit/v2/pkg/api/client"
)

func exposePortsRootless(ctx context.Context, rlkClient rlkclient.Client, ports []cni.PortMapping) error {
	return errors.New("cannot expose ports rootlessly on non-Linux hosts")
}

func unexposePortsRootless(ctx context.Context, rlkClient rlkclient.Client, ports []cni.PortMapping) error {
	return errors.New("cannot unexpose ports rootlessly on non-Linux hosts")
}
