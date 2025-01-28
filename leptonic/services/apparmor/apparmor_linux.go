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

	"go.farcloser.world/containers/security/apparmor"

	"github.com/containerd/nerdctl/v2/leptonic/services/helpers"
)

var (
	ErrServiceAppArmor           = errors.New("apparmor error")
	ErrServiceAppArmorCannotLoad = errors.New("unable to load apparmor profile - try with sudo")
)

func ListNames() ([]string, error) {
	profiles, err := apparmor.Profiles()
	if err != nil {
		return nil, errWrap(helpers.ErrConvert(err))
	}

	res := []string{}
	for _, f := range profiles {
		res = append(res, f.Name)
	}

	return res, nil
}

func List() ([]apparmor.Profile, error) {
	res, err := apparmor.Profiles()
	if err != nil {
		return nil, errWrap(helpers.ErrConvert(err))
	}

	return res, nil
}

func Inspect(profile string) (string, error) {
	res, err := apparmor.DumpDefaultProfile(profile)
	if err != nil {
		return "", errWrap(helpers.ErrConvert(err))
	}

	return res, nil
}

func Load(profile string) error {
	if !apparmor.CanLoadNewProfile() {
		return ErrServiceAppArmorCannotLoad
	}

	if err := apparmor.LoadDefaultProfile(profile); err != nil {
		return errWrap(helpers.ErrConvert(err))
	}

	return nil
}

func Unload(profile string) error {
	if err := apparmor.Unload(profile); err != nil {
		return errWrap(helpers.ErrConvert(err))
	}

	return nil
}

func errWrap(err error) error {
	return errors.Join(ErrServiceAppArmor, err)
}
