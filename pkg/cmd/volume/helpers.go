package volume

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/errdefs"
	"github.com/containerd/log"

	"github.com/farcloser/lepton/pkg/inspecttypes/dockercompat"
	"github.com/farcloser/lepton/pkg/inspecttypes/native"
	"github.com/farcloser/lepton/pkg/labels"
	"github.com/farcloser/lepton/pkg/mountutil"
)

func usedVolumes(ctx context.Context, containers []containerd.Container) (map[string]string, error) {
	usedVolumesList := make(map[string]string)
	for _, c := range containers {
		l, err := c.Labels(ctx)
		if err != nil {
			// Containerd note: there is no guarantee that the containers we got from the list still exist at this point
			// If that is the case, just ignore and move on
			if errors.Is(err, errdefs.ErrNotFound) {
				log.G(ctx).Debugf("container %q is gone - ignoring", c.ID())
				continue
			}
			return nil, err
		}
		mountsJSON, ok := l[labels.Mounts]
		if !ok {
			continue
		}

		var mounts []dockercompat.MountPoint
		err = json.Unmarshal([]byte(mountsJSON), &mounts)
		if err != nil {
			return nil, err
		}
		for _, m := range mounts {
			if m.Type == mountutil.Volume {
				usedVolumesList[m.Name] = l[labels.Name]
			}
		}
	}
	return usedVolumesList, nil
}

func filter(vols map[string]native.Volume, filters []string) (map[string]native.Volume, error) {
	labelFilterFuncs, nameFilterFuncs, sizeFilterFuncs, isFilter, err := getVolumeFilterFuncs(filters)
	if err != nil {
		return nil, err
	}
	if !isFilter {
		return vols, nil
	}
	for k, v := range vols {
		if !volumeMatchesFilter(v, labelFilterFuncs, nameFilterFuncs, sizeFilterFuncs) {
			delete(vols, k)
		}
	}
	return vols, nil
}

func getVolumeFilterFuncs(filters []string) ([]func(*map[string]string) bool, []func(string) bool, []func(int64) bool, bool, error) {
	isFilter := len(filters) > 0
	labelFilterFuncs := make([]func(*map[string]string) bool, 0)
	nameFilterFuncs := make([]func(string) bool, 0)
	sizeFilterFuncs := make([]func(int64) bool, 0)
	sizeOperators := []struct {
		Operand string
		Compare func(int64, int64) bool
	}{
		{">=", func(size, volumeSize int64) bool {
			return volumeSize >= size
		}},
		{"<=", func(size, volumeSize int64) bool {
			return volumeSize <= size
		}},
		{">", func(size, volumeSize int64) bool {
			return volumeSize > size
		}},
		{"<", func(size, volumeSize int64) bool {
			return volumeSize < size
		}},
		{"=", func(size, volumeSize int64) bool {
			return volumeSize == size
		}},
	}
	for _, filter := range filters {
		if strings.HasPrefix(filter, "name") || strings.HasPrefix(filter, "label") {
			subs := strings.SplitN(filter, "=", 2)
			if len(subs) < 2 {
				continue
			}
			switch subs[0] {
			case "name":
				nameFilterFuncs = append(nameFilterFuncs, func(name string) bool {
					return strings.Contains(name, subs[1])
				})
			case "label":
				v, k, hasValue := "", subs[1], false
				if subs := strings.SplitN(subs[1], "=", 2); len(subs) == 2 {
					hasValue = true
					k, v = subs[0], subs[1]
				}
				labelFilterFuncs = append(labelFilterFuncs, func(labels *map[string]string) bool {
					if labels == nil {
						return false
					}
					val, ok := (*labels)[k]
					if !ok || (hasValue && val != v) {
						return false
					}
					return true
				})
			}
			continue
		}
		if strings.HasPrefix(filter, "size") {
			for _, sizeOperator := range sizeOperators {
				if subs := strings.SplitN(filter, sizeOperator.Operand, 2); len(subs) == 2 {
					v, err := strconv.Atoi(subs[1])
					if err != nil {
						return nil, nil, nil, false, err
					}
					sizeFilterFuncs = append(sizeFilterFuncs, func(size int64) bool {
						return sizeOperator.Compare(int64(v), size)
					})
					break
				}
			}
			continue
		}
	}
	return labelFilterFuncs, nameFilterFuncs, sizeFilterFuncs, isFilter, nil
}

func volumeMatchesFilter(vol native.Volume, labelFilterFuncs []func(*map[string]string) bool, nameFilterFuncs []func(string) bool, sizeFilterFuncs []func(int64) bool) bool {
	for _, labelFilterFunc := range labelFilterFuncs {
		if !labelFilterFunc(vol.Labels) {
			return false
		}
	}
	for _, nameFilterFunc := range nameFilterFuncs {
		if !nameFilterFunc(vol.Name) {
			return false
		}
	}
	for _, sizeFilterFunc := range sizeFilterFuncs {
		if !sizeFilterFunc(vol.Size) {
			return false
		}
	}
	return true
}
