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
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"

	composecli "github.com/compose-spec/compose-go/v2/cli"
	compose "github.com/compose-spec/compose-go/v2/types"
	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/log"

	"go.farcloser.world/lepton/leptonic/identifiers"
	"go.farcloser.world/lepton/pkg/composer/serviceparser"
	"go.farcloser.world/lepton/pkg/reflectutil"
)

// Options groups the command line options recommended for a Compose implementation (ProjectOptions) and extra options
// for the cli
type Options struct {
	Project          string // empty for default
	ProjectDirectory string
	ConfigPaths      []string
	Profiles         []string
	Services         []string
	EnvFile          string
	CliCmd           string
	CliArgs          []string
	NetworkInUse     func(ctx context.Context, netName string) (bool, error)
	NetworkExists    func(string) (bool, error)
	VolumeExists     func(string) (bool, error)
	ImageExists      func(ctx context.Context, imageName string) (bool, error)
	EnsureImage      func(ctx context.Context, imageName, pullMode, platform string, ps *serviceparser.Service, quiet bool) error
	DebugPrintFull   bool // full debug print, may leak secret env var to logs
	Experimental     bool // enable experimental features
}

func New(o *Options, client *containerd.Client) (*Composer, error) {
	if o.CliCmd == "" {
		return nil, errors.New("got empty cmd")
	}
	if o.NetworkExists == nil || o.VolumeExists == nil || o.EnsureImage == nil {
		return nil, errors.New("got empty functions")
	}

	if o.Project != "" {
		if err := identifiers.Validate(o.Project); err != nil {
			return nil, fmt.Errorf("invalid project name: %w", err)
		}
	}

	var optionsFn []composecli.ProjectOptionsFn
	optionsFn = append(optionsFn,
		composecli.WithOsEnv,
		composecli.WithWorkingDirectory(o.ProjectDirectory),
	)
	if o.EnvFile != "" {
		optionsFn = append(optionsFn,
			composecli.WithEnvFiles(o.EnvFile),
		)
	}
	optionsFn = append(optionsFn,
		composecli.WithConfigFileEnv,
		composecli.WithDefaultConfigPath,
		composecli.WithEnvFiles(),
		composecli.WithDotEnv,
		composecli.WithName(o.Project),
	)

	projectOptions, err := composecli.NewProjectOptions(o.ConfigPaths, optionsFn...)
	if err != nil {
		return nil, err
	}
	project, err := projectOptions.LoadProject(context.TODO())
	if err != nil {
		return nil, err
	}

	if len(o.Services) > 0 {
		s, err := project.GetServices(o.Services...)
		if err != nil {
			return nil, err
		}
		o.Profiles = append(o.Profiles, s.GetProfiles()...)
	}

	project, err = project.WithProfiles(o.Profiles)
	if err != nil {
		return nil, err
	}

	if o.DebugPrintFull {
		projectJSON, err := json.MarshalIndent(project, "", "    ")
		if err != nil {
			return nil, err
		}
		log.L.Debug("printing project JSON")
		log.L.Debugf("%s", projectJSON)
	}

	if unknown := reflectutil.UnknownNonEmptyFields(project,
		"Name",
		"WorkingDir",
		"Environment",
		"Services",
		"Networks",
		"Volumes",
		"Secrets",
		"Configs",
		"ComposeFiles"); len(unknown) > 0 {
		log.L.Warnf("Ignoring: %+v", unknown)
	}

	c := &Composer{
		Options: *o,
		project: project,
		client:  client,
	}

	return c, nil
}

type Composer struct {
	Options
	project *compose.Project
	client  *containerd.Client
}

func (c *Composer) createCliCmd(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, c.CliCmd, append(c.CliArgs, args...)...)
}

func (c *Composer) runCliCmd(ctx context.Context, args ...string) error {
	cmd := c.createCliCmd(ctx, args...)
	if c.DebugPrintFull {
		log.G(ctx).Debugf("Running %v", cmd.Args)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error while executing %v: %q: %w", cmd.Args, string(out), err)
	}
	return nil
}

// Services returns the parsed Service objects in dependency order.
func (c *Composer) Services(ctx context.Context, svcs ...string) ([]*serviceparser.Service, error) {
	var services []*serviceparser.Service

	if err := c.project.ForEachService(svcs, func(name string, svc *compose.ServiceConfig) error {
		parsed, err := serviceparser.Parse(c.project, *svc)
		if err != nil {
			return err
		}
		services = append(services, parsed)
		return nil
	}); err != nil {
		return nil, err
	}
	return services, nil
}

// ServiceNames returns service names in dependency order.
func (c *Composer) ServiceNames(svcs ...string) ([]string, error) {
	var names []string
	if err := c.project.ForEachService(svcs, func(name string, svc *compose.ServiceConfig) error {
		names = append(names, svc.Name)
		return nil
	}); err != nil {
		return nil, err
	}
	return names, nil
}
