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

package composer

import (
	"context"
	"fmt"

	"github.com/containerd/log"

	"go.farcloser.world/lepton/pkg/labels"
	"go.farcloser.world/lepton/pkg/reflectutil"
)

func (c *Composer) upVolume(ctx context.Context, shortName string) error {
	vol, ok := c.project.Volumes[shortName]
	if !ok {
		return fmt.Errorf("invalid volume name %q", shortName)
	}
	if vol.External {
		// NOP
		return nil
	}

	if unknown := reflectutil.UnknownNonEmptyFields(&vol, "Name"); len(unknown) > 0 {
		log.G(ctx).Warnf("Ignoring: volume %s: %+v", shortName, unknown)
	}

	// shortName is like "db_data", fullName is like "compose-wordpress_db_data"
	fullName := vol.Name
	// FIXME: this is racy. By the time we get below to creating the volume, there is no guarantee that things are still
	// fine.
	// Furthermore, by the time we are done creating all the volumes, they may very well have been destroyed.
	// This cannot be fixed without getting rid of the whole "shell-out" approach entirely.
	volExists, err := c.VolumeExists(fullName)
	if err != nil {
		return err
	} else if !volExists {
		log.G(ctx).Infof("Creating volume %s", fullName)
		// add metadata labels to volume https://github.com/compose-spec/compose-spec/blob/master/spec.md#labels-2
		createArgs := []string{
			fmt.Sprintf("--label=%s=%s", labels.ComposeProject, c.project.Name),
			fmt.Sprintf("--label=%s=%s", labels.ComposeVolume, shortName),
			fullName,
		}
		if err := c.runCliCmd(ctx, append([]string{"volume", "create"}, createArgs...)...); err != nil {
			return err
		}
	}
	return nil
}
