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

package compose

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/snapshots"
	"github.com/containerd/containerd/v2/pkg/progress"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/leptonic/services/containerd"
	"go.farcloser.world/lepton/pkg/cmd/compose"
	"go.farcloser.world/lepton/pkg/formatter"
	"go.farcloser.world/lepton/pkg/imgutil"
	"go.farcloser.world/lepton/pkg/labels"
	"go.farcloser.world/lepton/pkg/strutil"
)

func imagesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "images [flags] [SERVICE...]",
		Short:         "List images used by created containers in services",
		RunE:          imagesAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().String("format", "", "Format the output. Supported values: [json]")
	cmd.Flags().BoolP("quiet", "q", false, "Only show numeric image IDs")

	return cmd
}

func imagesAction(cmd *cobra.Command, args []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}

	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return err
	}
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return err
	}
	if format != formatter.FormatJSON && format != "" {
		return fmt.Errorf("unsupported format %s, supported formats are: [json]", format)
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()

	options, err := getComposeOptions(cmd, globalOptions.DebugFull, globalOptions.Experimental)
	if err != nil {
		return err
	}
	c, err := compose.New(cli, globalOptions, options, cmd.OutOrStdout(), cmd.ErrOrStderr())
	if err != nil {
		return err
	}

	serviceNames, err := c.ServiceNames(args...)
	if err != nil {
		return err
	}

	containers, err := c.Containers(ctx, serviceNames...)
	if err != nil {
		return err
	}

	if quiet {
		return printComposeImageIDs(ctx, containers)
	}

	sn := cli.SnapshotService(globalOptions.Snapshotter)

	return printComposeImages(ctx, cmd, containers, sn, format)
}

func printComposeImageIDs(ctx context.Context, containers []client.Container) error {
	ids := []string{}
	for _, c := range containers {
		image, err := c.Image(ctx)
		if err != nil {
			return err
		}
		metaImage := image.Metadata()
		id := metaImage.Target.Digest.String()
		if !strutil.InStringSlice(ids, id) {
			ids = append(ids, id)
		}
	}

	for _, id := range ids {
		// always truncate image ids.
		fmt.Fprintln(os.Stdout, strings.Split(id, ":")[1][:12])
	}
	return nil
}

func printComposeImages(
	ctx context.Context,
	cmd *cobra.Command,
	containers []client.Container,
	sn snapshots.Snapshotter,
	format string,
) error {
	type composeImagePrintable struct {
		ContainerName string
		Repository    string
		Tag           string
		ImageID       string
		Size          string
	}

	imagePrintables := make([]composeImagePrintable, len(containers))
	eg, ctx := errgroup.WithContext(ctx)
	for i, c := range containers {
		eg.Go(func() error {
			info, err := c.Info(ctx, client.WithoutRefreshedMetadata)
			if err != nil {
				return err
			}
			containerName := info.Labels[labels.Name]

			image, err := c.Image(ctx)
			if err != nil {
				return err
			}

			size, err := imgutil.UnpackedImageSize(ctx, sn, image)
			if err != nil {
				return err
			}

			metaImage := image.Metadata()
			repository, tag := imgutil.ParseRepoTag(metaImage.Name)
			imageID := metaImage.Target.Digest.String()
			if repository == "" {
				repository = "<none>"
			}
			if tag == "" {
				tag = "<none>"
			}
			if format != formatter.FormatJSON {
				imageID = strings.Split(imageID, ":")[1][:12]
			}

			// no race condition since each goroutine accesses different `i`
			imagePrintables[i] = composeImagePrintable{
				ContainerName: containerName,
				Repository:    repository,
				Tag:           tag,
				ImageID:       imageID,
				Size:          progress.Bytes(size).String(),
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	if format == formatter.FormatJSON {
		outJSON, err := formatter.ToJSON(imagePrintables, "", "")
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(cmd.OutOrStdout(), outJSON)
		return err
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 4, 8, 4, ' ', 0)
	fmt.Fprintln(w, "Container\tRepository\tTag\tImage Id\tSize")
	for _, p := range imagePrintables {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			p.ContainerName,
			p.Repository,
			p.Tag,
			p.ImageID,
			p.Size,
		); err != nil {
			return err
		}
	}

	return w.Flush()
}
