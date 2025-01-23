/*
   Copyright The containerd Authors.

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

package logging

import (
	"bytes"
	"context"
	"io"
	"os"
	"text/template"

	"github.com/docker/cli/templates"

	"github.com/containerd/containerd/v2/core/runtime/v2/logging"
	"github.com/containerd/log"

	"github.com/containerd/nerdctl/v2/leptonic/journald"
	"github.com/containerd/nerdctl/v2/pkg/clientutil"
	"github.com/containerd/nerdctl/v2/pkg/containerutil"
	"github.com/containerd/nerdctl/v2/pkg/strutil"
)

var JournalDriverLogOpts = []string{
	Tag,
}

func JournalLogOptsValidate(logOptMap map[string]string) error {
	for key := range logOptMap {
		if !strutil.InStringSlice(JournalDriverLogOpts, key) {
			log.L.Warnf("log-opt %s is ignored for journald log driver", key)
		}
	}
	return nil
}

type JournaldLogger struct {
	Opts    map[string]string
	vars    map[string]string
	Address string
}

type identifier struct {
	ID        string
	FullID    string
	Namespace string
}

func (journaldLogger *JournaldLogger) Init(dataStore, ns, id string) error {
	return nil
}

func (journaldLogger *JournaldLogger) PreProcess(ctx context.Context, dataStore string, config *logging.Config) error {
	if err := journald.Init(); err != nil {
		return err
	}
	shortID := config.ID[:12]
	var syslogIdentifier string
	syslogIdentifier = shortID
	if tag, ok := journaldLogger.Opts[Tag]; ok {
		var tmpl *template.Template
		var err error
		tmpl, err = templates.Parse(tag)
		if err != nil {
			return err
		}

		if tmpl != nil {
			idn := identifier{
				ID:        shortID,
				FullID:    config.ID,
				Namespace: config.Namespace,
			}
			var b bytes.Buffer
			if err := tmpl.Execute(&b, idn); err != nil {
				return err
			}
			syslogIdentifier = b.String()
		}
	}

	client, ctx, cancel, err := clientutil.NewClient(ctx, config.Namespace, journaldLogger.Address)
	if err != nil {
		return err
	}
	defer func() {
		cancel()
		client.Close()
	}()
	containerID := config.ID
	container, err := client.LoadContainer(ctx, containerID)
	if err != nil {
		return err
	}
	containerLabels, err := container.Labels(ctx)
	if err != nil {
		return err
	}
	containerInfo, err := container.Info(ctx)
	if err != nil {
		return err
	}

	// construct log metadata for the container
	journaldLogger.vars = map[string]string{
		"SYSLOG_IDENTIFIER":   syslogIdentifier,
		"CONTAINER_TAG":       syslogIdentifier,
		"CONTAINER_ID":        shortID,
		"CONTAINER_ID_FULL":   containerID,
		"CONTAINER_NAME":      containerutil.GetContainerName(containerLabels),
		"CONTAINER_NAMESPACE": config.Namespace,
		"IMAGE_NAME":          containerInfo.Image,
	}

	return nil
}

func (journaldLogger *JournaldLogger) Process(stdout <-chan string, stderr <-chan string) error {
	return journald.WriteLogs(journaldLogger.vars, stdout, stderr)
}

func (journaldLogger *JournaldLogger) PostProcess() error {
	return nil
}

// ViewLogs formats command line arguments for `journalctl` with the provided log viewing options and
// exec's and redirects `journalctl`s outputs to the provided io.Writers.
func viewLogsJournald(lvopts LogViewOptions, stdout, stderr io.Writer, stopChannel chan os.Signal) error {
	return journald.ReadLogs(journald.LogViewOptions{
		ID:     lvopts.ContainerID,
		Follow: lvopts.Follow,
		Since:  lvopts.Since,
		Until:  lvopts.Until,
		NoTail: lvopts.Tail == 0,
	}, stdout, stderr, stopChannel)
}
