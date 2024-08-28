package volume

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"text/tabwriter"
	"text/template"

	"github.com/containerd/containerd/v2/pkg/progress"
	"github.com/containerd/log"
	"github.com/spf13/cobra"

	"github.com/farcloser/lepton/cmd/lepton/helpers"
	"github.com/farcloser/lepton/pkg/api/types"
	"github.com/farcloser/lepton/pkg/cmd/volume"
	"github.com/farcloser/lepton/pkg/formatter"
	"github.com/farcloser/lepton/pkg/inspecttypes/native"
)

type Formats string

const Raw Formats = "raw"
const JSON Formats = "json"
const Table Formats = "table"
const Wide Formats = "wide"

func NewVolumeLsCommand() *cobra.Command {
	volumeLsCommand := &cobra.Command{
		Use:           "ls",
		Aliases:       []string{"list"},
		Short:         "List volumes",
		RunE:          volumeLsAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	volumeLsCommand.Flags().BoolP(flagQuiet, "q", false, "Only display volume names")
	// Alias "-f" is reserved for "--filter"
	volumeLsCommand.Flags().String(flagFormat, "", "Format the output using the given go template")
	volumeLsCommand.Flags().StringSliceP(flagFilter, "f", []string{}, "Filter matches volumes based on given conditions")
	volumeLsCommand.Flags().BoolP(flagSize, "s", false, "Display the disk usage of volumes. Can be slow with volumes having loads of directories.")

	err := volumeLsCommand.RegisterFlagCompletionFunc(flagFormat, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{string(JSON), string(Table), string(Wide)}, cobra.ShellCompDirectiveNoFileComp
	})

	if err != nil {
		log.L.Errorf("Failed to register format completion function: %v", err)
	}

	return volumeLsCommand
}

func processVolumeLsOptions(cmd *cobra.Command) (*types.VolumeListOptions, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return nil, err
	}

	quiet, err := cmd.Flags().GetBool(flagQuiet)
	if err != nil {
		return nil, err
	}

	format, err := cmd.Flags().GetString(flagFormat)
	if err != nil {
		return nil, err
	}

	size, err := cmd.Flags().GetBool(flagSize)
	if err != nil {
		return nil, err
	}

	filters, err := cmd.Flags().GetStringSlice(flagFilter)
	if err != nil {
		return nil, err
	}

	return &types.VolumeListOptions{
		GOptions: globalOptions,
		Quiet:    quiet,
		Format:   format,
		Size:     size,
		Filters:  filters,
	}, nil
}

func volumeLsAction(cmd *cobra.Command, args []string) error {
	options, err := processVolumeLsOptions(cmd)
	if err != nil {
		return err
	}

	vols, err := volume.List(options)
	if err != nil {
		return err
	}

	return lsPrintOutput(vols, cmd.OutOrStdout(), options)
}

type volumePrintable struct {
	Driver     string
	Labels     string
	Mountpoint string
	Name       string
	Scope      string
	Size       string
	// TODO: "Links"
}

func lsPrintOutput(vols map[string]native.Volume, w io.Writer, options *types.VolumeListOptions) error {
	var tmpl *template.Template
	switch options.Format {
	case "", string(Table), string(Wide):
		w = tabwriter.NewWriter(w, 4, 8, 4, ' ', 0)
		if !options.Quiet {
			if options.Size {
				_, err := fmt.Fprintln(w, "VOLUME NAME\tDIRECTORY\tSIZE")
				if err != nil {
					return err
				}
			} else {
				_, err := fmt.Fprintln(w, "VOLUME NAME\tDIRECTORY")
				if err != nil {
					return err
				}
			}
		}
	case string(Raw):
		return errors.New("unsupported format: \"raw\"")
	default:
		if options.Quiet {
			return errors.New("format and quiet must not be specified together")
		}
		var err error
		tmpl, err = formatter.ParseTemplate(options.Format)
		if err != nil {
			return err
		}
	}

	for _, v := range vols {
		p := volumePrintable{
			Driver:     "local",
			Labels:     "",
			Mountpoint: v.Mountpoint,
			Name:       v.Name,
			Scope:      "local",
		}
		if v.Labels != nil {
			p.Labels = formatter.FormatLabels(*v.Labels)
		}
		if options.Size {
			p.Size = progress.Bytes(v.Size).String()
		}
		if tmpl != nil {
			var b bytes.Buffer
			if err := tmpl.Execute(&b, p); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(w, b.String()); err != nil {
				return err
			}
		} else if options.Quiet {
			_, err := fmt.Fprintln(w, p.Name)
			if err != nil {
				return err
			}
		} else if options.Size {
			_, err := fmt.Fprintf(w, "%s\t%s\t%s\n", p.Name, p.Mountpoint, p.Size)
			if err != nil {
				return err
			}
		} else {
			_, err := fmt.Fprintf(w, "%s\t%s\n", p.Name, p.Mountpoint)
			if err != nil {
				return err
			}
		}
	}
	if f, ok := w.(formatter.Flusher); ok {
		return f.Flush()
	}
	return nil
}
