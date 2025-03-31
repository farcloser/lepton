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

package container

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/log"

	"go.farcloser.world/containers/reference"

	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/idutil/containerwalker"
	"go.farcloser.world/lepton/pkg/imgutil/commit"
)

// Commit will commit a container’s file changes or settings into a new image.
func Commit(
	ctx context.Context,
	client *containerd.Client,
	rawRef string,
	req string,
	options options.ContainerCommit,
) error {
	parsedReference, err := reference.Parse(rawRef)
	if err != nil {
		return err
	}

	changes, err := parseChanges(options.Change)
	if err != nil {
		return err
	}

	opts := &commit.Opts{
		Author:  options.Author,
		Message: options.Message,
		Ref:     parsedReference.String(),
		Pause:   options.Pause,
		Changes: changes,
	}

	walker := &containerwalker.ContainerWalker{
		Client: client,
		OnFound: func(ctx context.Context, found containerwalker.Found) error {
			if found.MatchCount > 1 {
				return fmt.Errorf("multiple IDs found with provided prefix: %s", found.Req)
			}
			imageID, err := commit.Commit(ctx, client, found.Container, opts, options.GOptions)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(options.Stdout, imageID)
			return err
		},
	}
	n, err := walker.Walk(ctx, req)
	if err != nil {
		return err
	} else if n == 0 {
		return fmt.Errorf("no such container %s", req)
	}
	return nil
}

func parseChanges(userChanges []string) (commit.Changes, error) {
	const (
		// XXX: Where can I get a constants for this?
		commandDirective    = "CMD"
		entrypointDirective = "ENTRYPOINT"
	)
	if userChanges == nil {
		return commit.Changes{}, nil
	}
	var changes commit.Changes
	for _, change := range userChanges {
		if change == "" {
			return commit.Changes{}, errors.New("received an empty value in change flag")
		}
		changeFields := strings.Fields(change)

		switch changeFields[0] {
		case commandDirective:
			var overrideCMD []string
			if err := json.Unmarshal([]byte(change[len(changeFields[0]):]), &overrideCMD); err != nil {
				return commit.Changes{}, fmt.Errorf("malformed json in change flag value %q", change)
			}
			if changes.CMD != nil {
				log.L.Warn("multiple change flags supplied for the CMD directive, overriding with last supplied")
			}
			changes.CMD = overrideCMD
		case entrypointDirective:
			var overrideEntrypoint []string
			if err := json.Unmarshal([]byte(change[len(changeFields[0]):]), &overrideEntrypoint); err != nil {
				return commit.Changes{}, fmt.Errorf("malformed json in change flag value %q", change)
			}
			if changes.Entrypoint != nil {
				log.L.Warnf(
					"multiple change flags supplied for the Entrypoint directive, overriding with last supplied",
				)
			}
			changes.Entrypoint = overrideEntrypoint
		default: // TODO: Support the rest of the change directives
			return commit.Changes{}, fmt.Errorf("unknown change directive %q", changeFields[0])
		}
	}
	return changes, nil
}
