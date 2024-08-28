package volume

import (
	"errors"

	"github.com/farcloser/lepton/pkg/clientutil"
	"github.com/farcloser/lepton/pkg/errs"
	"github.com/farcloser/lepton/pkg/mountutil/volumestore"
)

func VolumesNames(dataRoot string, address string, namespace string) ([]string, error) {
	dataStore, err := clientutil.DataStore(dataRoot, address)
	if err != nil {
		return nil, errors.Join(errs.ErrSystemFailure, err)
	}

	volumeStore, err := volumestore.New(dataStore, namespace)
	if err != nil {
		return nil, errors.Join(errs.ErrSystemFailure, err)
	}

	volumes, err := volumeStore.ListNames()
	if err != nil {
		return nil, err
	}

	return volumes, nil
}
