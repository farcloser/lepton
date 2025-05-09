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
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"text/tabwriter"
	"text/template"
	"time"

	eventstypes "github.com/containerd/containerd/api/events"
	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/events"
	"github.com/containerd/errdefs"
	"github.com/containerd/log"
	"github.com/containerd/typeurl/v2"

	"go.farcloser.world/containers/security/cgroups"
	stats2 "go.farcloser.world/containers/stats"

	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/clientutil"
	"go.farcloser.world/lepton/pkg/containerutil"
	"go.farcloser.world/lepton/pkg/eventutil"
	"go.farcloser.world/lepton/pkg/formatter"
	"go.farcloser.world/lepton/pkg/idutil/containerwalker"
	"go.farcloser.world/lepton/pkg/rootlessutil"
	"go.farcloser.world/lepton/pkg/statsutil"
)

type statsStruct struct {
	mu sync.Mutex
	cs []*stats2.Stats
}

// add is from
// https://github.com/docker/cli/blob/3fb4fb83dfb5db0c0753a8316f21aea54dab32c5/cli/command/container/stats_helpers.go#L26-L34
func (s *statsStruct) add(cs *stats2.Stats) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.isKnownContainer(cs.ID); !exists {
		s.cs = append(s.cs, cs)
		return true
	}
	return false
}

// remove is from
// https://github.com/docker/cli/blob/3fb4fb83dfb5db0c0753a8316f21aea54dab32c5/cli/command/container/stats_helpers.go#L36-L42
func (s *statsStruct) remove(id string) {
	s.mu.Lock()
	if i, exists := s.isKnownContainer(id); exists {
		s.cs = append(s.cs[:i], s.cs[i+1:]...)
	}
	s.mu.Unlock()
}

// isKnownContainer is from
// https://github.com/docker/cli/blob/3fb4fb83dfb5db0c0753a8316f21aea54dab32c5/cli/command/container/stats_helpers.go#L44-L51
func (s *statsStruct) isKnownContainer(cid string) (int, bool) {
	for i, c := range s.cs {
		if c.ID == cid {
			return i, true
		}
	}
	return -1, false
}

// Stats displays a live stream of container(s) resource usage statistics.
func Stats(
	ctx context.Context,
	client *containerd.Client,
	containerIDs []string,
	options options.ContainerStats,
) error {
	if rootlessutil.IsRootless() && cgroups.Version() < 2 {
		return errors.New(
			"stats requires cgroup v2 for rootless containers, see https://rootlesscontaine.rs/getting-started/common/cgroup2/",
		)
	}

	showAll := len(containerIDs) == 0
	closeChan := make(chan error)

	var err error
	w := options.Stdout
	var tmpl *template.Template
	switch options.Format {
	case formatter.FormatNone, formatter.FormatTable:
		w = tabwriter.NewWriter(options.Stdout, 10, 1, 3, ' ', 0)
	default:
		tmpl, err = formatter.ParseTemplate(options.Format)
		if err != nil {
			return err
		}
	}

	// waitFirst is a WaitGroup to wait first stat data's reach for each container
	waitFirst := &sync.WaitGroup{}
	cStats := statsStruct{}

	monitorContainerEvents := func(started chan<- struct{}, c chan *events.Envelope) {
		eventsClient := client.EventService()
		eventsCh, errCh := eventsClient.Subscribe(ctx)

		// Whether we successfully subscribed to eventsCh or not, we can now
		// unblock the main goroutine.
		close(started)

		for {
			select {
			case event := <-eventsCh:
				c <- event
			case err = <-errCh:
				closeChan <- err
				return
			}
		}
	}

	// getContainerList get all existing containers (only used when calling `stats` without arguments).
	getContainerList := func() {
		containers, err := client.Containers(ctx)
		if err != nil {
			closeChan <- err
		}

		for _, c := range containers {
			cStatus := formatter.ContainerStatus(ctx, c)
			if !options.All {
				if !strings.HasPrefix(cStatus, "Up") {
					continue
				}
			}
			s := stats2.NewStats(c.ID())
			if cStats.add(s) {
				waitFirst.Add(1)
				go collect(ctx, options.GOptions, s, waitFirst, c.ID(), !options.NoStream)
			}
		}
	}

	if showAll {
		started := make(chan struct{})
		var (
			datacc *eventstypes.ContainerCreate
			datacd *eventstypes.ContainerDelete
		)

		eh := eventutil.InitEventHandler()
		eh.Handle("/containers/create", func(e events.Envelope) {
			if e.Event != nil {
				anydata, err := typeurl.UnmarshalAny(e.Event)
				if err != nil {
					// just skip
					return
				}
				switch v := anydata.(type) {
				case *eventstypes.ContainerCreate:
					datacc = v
				default:
					// just skip
					return
				}
			}
			s := stats2.NewStats(datacc.ID)
			if cStats.add(s) {
				waitFirst.Add(1)
				go collect(ctx, options.GOptions, s, waitFirst, datacc.ID, !options.NoStream)
			}
		})

		eh.Handle("/containers/delete", func(e events.Envelope) {
			if e.Event != nil {
				anydata, err := typeurl.UnmarshalAny(e.Event)
				if err != nil {
					// just skip
					return
				}
				switch v := anydata.(type) {
				case *eventstypes.ContainerDelete:
					datacd = v
				default:
					// just skip
					return
				}
			}
			cStats.remove(datacd.ID)
		})

		eventChan := make(chan *events.Envelope)

		go eh.Watch(eventChan)
		go monitorContainerEvents(started, eventChan)

		defer close(eventChan)
		<-started

		// Start a goroutine to retrieve the initial list of containers stats.
		getContainerList()

		// make sure each container get at least one valid stat data
		waitFirst.Wait()

	} else {
		walker := &containerwalker.ContainerWalker{
			Client: client,
			OnFound: func(ctx context.Context, found containerwalker.Found) error {
				s := stats2.NewStats(found.Container.ID())
				if cStats.add(s) {
					waitFirst.Add(1)
					go collect(ctx, options.GOptions, s, waitFirst, found.Container.ID(), !options.NoStream)
				}
				return nil
			},
		}

		if err := walker.WalkAll(ctx, containerIDs, false); err != nil {
			return err
		}

		// make sure each container get at least one valid stat data
		waitFirst.Wait()

	}

	cleanScreen := func() {
		if !options.NoStream {
			fmt.Fprint(options.Stdout, "\033[2J")
			fmt.Fprint(options.Stdout, "\033[H")
		}
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	// firstTick is for creating distant CPU readings.
	// firstTick stats are not displayed.
	firstTick := true
	for range ticker.C {
		cleanScreen()
		ccstats := []stats2.Entry{}
		cStats.mu.Lock()
		for _, c := range cStats.cs {
			if err := c.GetError(); err != nil {
				fmt.Fprintf(options.Stderr, "unable to get stat entry: %s\n", err)
			}
			ccstats = append(ccstats, c.GetStatistics())
		}
		cStats.mu.Unlock()

		if !firstTick {
			// print header for every tick
			if options.Format == formatter.FormatNone || options.Format == formatter.FormatTable {
				fmt.Fprintln(w, "CONTAINER ID\tNAME\tCPU %\tMEM USAGE / LIMIT\tMEM %\tNET I/O\tBLOCK I/O\tPIDS")
			}
		}

		for _, c := range ccstats {
			if c.ID == "" {
				continue
			}
			rc := statsutil.RenderEntry(&c)
			if !firstTick {
				if tmpl != nil {
					var b bytes.Buffer
					if err := tmpl.Execute(&b, rc); err != nil {
						break
					}
					if _, err = fmt.Fprintln(options.Stdout, b.String()); err != nil {
						break
					}
				} else {
					if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
						rc.ID(options.NoTrunc),
						rc.Name(options.NoTrunc),
						rc.CPUPerc(),
						rc.MemUsage(),
						rc.MemPerc(),
						rc.NetIO(),
						rc.BlockIO(),
						rc.PIDs(),
					); err != nil {
						break
					}
				}
			}
		}
		if f, ok := w.(formatter.Flusher); ok {
			f.Flush()
		}

		if len(cStats.cs) == 0 && !showAll {
			break
		}
		if options.NoStream && !firstTick {
			break
		}
		select {
		case err, ok := <-closeChan:
			if ok {
				if err != nil {
					return err
				}
			}
		default:
			// just skip
		}
		firstTick = false
	}

	return err
}

func collect(
	ctx context.Context,
	globalOptions *options.Global,
	s *stats2.Stats,
	waitFirst *sync.WaitGroup,
	id string,
	_noStream bool,
) {
	log.G(ctx).Debugf("collecting stats for %s", s.ID)
	var (
		getFirst = true
		u        = make(chan error, 1)
	)

	defer func() {
		// if error happens, and we get nothing of stats, release wait group whatever
		if getFirst {
			getFirst = false
			waitFirst.Done()
		}
	}()
	client, ctx, cancel, err := clientutil.NewClient(ctx, globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		s.SetError(err)
		return
	}
	defer func() {
		cancel()
		client.Close()
	}()
	container, err := client.LoadContainer(ctx, id)
	if err != nil {
		s.SetError(err)
		return
	}

	go func() {
		previousStats := new(stats2.ContainerStats)
		firstSet := true
		for {
			labels, err := container.Labels(ctx)
			if err != nil {
				u <- err
				continue
			}

			// when (firstSet == true), we only set container stats without rendering stat entry
			statsEntry, err := setContainerStatsAndRenderStatsEntry(ctx, container, previousStats)
			if err != nil {
				u <- err
				continue
			}

			statsEntry.Name = containerutil.GetContainerName(labels)

			if firstSet {
				firstSet = false
			} else {
				s.SetStatistics(statsEntry)
			}
			u <- nil
			// sleep to create distant CPU readings
			time.Sleep(500 * time.Millisecond)
		}
	}()
	for {
		select {
		case <-time.After(6 * time.Second):
			// zero out the values if we have not received an update within
			// the specified duration.
			s.SetErrorAndReset(errors.New("timeout waiting for stats"))
			// if this is the first stat you get, release WaitGroup
			if getFirst {
				getFirst = false
				waitFirst.Done()
			}
		case err := <-u:
			if err != nil {
				if !errdefs.IsNotFound(err) {
					s.SetError(err)
					continue
				}
			}
			// if this is the first stat you get, release WaitGroup
			if getFirst {
				getFirst = false
				waitFirst.Done()
			}
		}
	}
}
