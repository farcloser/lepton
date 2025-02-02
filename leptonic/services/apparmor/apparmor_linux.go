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

package apparmor

import (
	"errors"

	"github.com/containerd/containerd/v2/pkg/oci"

	"go.farcloser.world/containers/security/apparmor"

	"go.farcloser.world/lepton/leptonic/errs"
	"go.farcloser.world/lepton/leptonic/services/helpers"
	"go.farcloser.world/lepton/pkg/rootlessutil"
)

const (
	Empty      = ""
	Unconfined = "unconfined"
)

var (
	ErrServiceAppArmor    = errors.New("apparmor service error")
	ErrUnsupported        = errors.New("it does not seem like apparmor is enabled on the host")
	ErrCannotLoadOrUnload = errors.New("not enough permissions to load or unload profiles")
	ErrCannotApply        = errors.New("requested profile cannot be applied (you should check if it is loaded)")
)

func ListNames() ([]string, error) {
	profiles, err := apparmor.Profiles()
	if err != nil {
		return nil, errWrap(helpers.ErrConvert(err), ErrServiceAppArmor, errs.ErrSystemFailure)
	}

	res := []string{}
	for _, f := range profiles {
		res = append(res, f.Name)
	}

	return res, nil
}

func List() ([]*apparmor.Profile, error) {
	res, err := apparmor.Profiles()

	return res, errWrap(helpers.ErrConvert(err), ErrServiceAppArmor, errs.ErrSystemFailure)
}

func Inspect(asName string) (string, error) {
	res, err := apparmor.DumpCurrentProfileAs(asName)

	return res, errWrap(helpers.ErrConvert(err), ErrServiceAppArmor, errs.ErrSystemFailure)
}

func Load(asName string) error {
	if !apparmor.CanLoadProfile() {
		return errWrap(ErrCannotLoadOrUnload, ErrServiceAppArmor)
	}

	return errWrap(helpers.ErrConvert(apparmor.LoadDefaultProfileAs(asName)), ErrServiceAppArmor, errs.ErrSystemFailure)
}

func Unload(profileName string) error {
	if !apparmor.CanLoadProfile() {
		return errWrap(ErrCannotLoadOrUnload, ErrServiceAppArmor)
	}

	return errWrap(helpers.ErrConvert(apparmor.UnloadProfile(profileName)), ErrServiceAppArmor)
}

func GetSpecOptions(securityOpt string) (oci.SpecOpts, error) {
	// If opt is the empty string, that is an error
	if securityOpt == Empty {
		return nil, errWrap(errors.New("security-opt \"apparmor\" can't be set to the empty string"), ErrServiceAppArmor, errs.ErrInvalidArgument)
	}

	// If unconfined, just return
	if securityOpt == Unconfined {
		return nil, nil
	}

	if !apparmor.Enabled() {
		return nil, errWrap(ErrUnsupported, ErrServiceAppArmor)
	}

	// Otherwise, if we can load, go for it
	if apparmor.CanLoadProfile() {
		if err := apparmor.LoadDefaultProfileAs(securityOpt); err != nil {
			return nil, errWrap(helpers.ErrConvert(err), ErrServiceAppArmor)
		}
	}

	// Either way, if we can, pass it along, and hard error otherwise
	if !apparmor.CanApplyProfile(securityOpt) {
		return nil, errWrap(ErrCannotApply, ErrServiceAppArmor)
	}

	return apparmor.WithProfile(securityOpt), nil
}

func GetInfo(withName string) (bool, error) {
	enabled := false
	if apparmor.Enabled() {
		enabled = true
		if rootlessutil.IsRootless() && !apparmor.CanApplyProfile(withName) {
			return enabled, ErrCannotApply
		}
	}

	return enabled, nil
}

func errWrap(err error, wrappers ...error) error {
	if err != nil {
		return errors.Join(append(wrappers, err)...)
	}

	return nil
}
