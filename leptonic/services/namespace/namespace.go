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

package namespace

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/identifiers"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/containerd/errdefs"

	"github.com/containerd/nerdctl/v2/leptonic/api"
	"github.com/containerd/nerdctl/v2/leptonic/errs"
)

/*
Notes:
Containerd implementation is broken:
- identifier validation is inconsistent
- errors being returned are inconsistent
- Labels always happily returns, regardless of whether the namespace exists or not

Hence:
- we validate names first
- inspect compares to the full list
*/

var (
	ErrServiceNamespace = errors.New("namespace error")
)

// FIXME will probably have to share this across
func validate(name string) error {
	if err := identifiers.Validate(name); err != nil {
		return errors.Join(errs.ErrInvalidArgument, err)
	}

	return nil
}

// FIXME should this one be in a different, shared package?
func NamespacedContext(ctx context.Context, name string) context.Context {
	return namespaces.WithNamespace(ctx, name)
}

func List(ctx context.Context, cli *client.Client) ([]string, error) {
	service := cli.NamespaceService()

	list, err := service.List(ctx)
	if err != nil {
		return nil, errWrap(errConvert(err))
	}

	return list, nil
}

func Create(ctx context.Context, cli *client.Client, name string, labels map[string]string) error {
	if err := validate(name); err != nil {
		return errWrap(err)
	}

	service := cli.NamespaceService()

	if err := service.Create(ctx, name, labels); err != nil {
		return errWrap(err)
	}

	return nil
}

func Update(ctx context.Context, cli *client.Client, name string, labels map[string]string) []error {
	if err := validate(name); err != nil {
		return []error{errWrap(err)}
	}

	if len(labels) == 0 {
		return []error{errWrap(errs.ErrInvalidArgument)}
	}

	service := cli.NamespaceService()
	resultErrors := []error{}

	list, err := service.List(ctx)
	if err != nil {
		return []error{errWrap(errConvert(err))}
	}

	if !slices.Contains(list, name) {
		return append(resultErrors, errWrap(errs.ErrNotFound))
	}

	for k, v := range labels {
		err = update(ctx, service, name, k, v)
		if err != nil {
			resultErrors = append(resultErrors, errWrap(err))
			continue
		}
	}

	return resultErrors
}

func Inspect(ctx context.Context, cli *client.Client, names []string) ([]*api.Namespace, []error) {
	service := cli.NamespaceService()
	resultErrors := []error{}
	result := []*api.Namespace{}

	list, err := service.List(ctx)
	if err != nil {
		return nil, []error{errWrap(errConvert(err))}
	}

	for _, name := range names {
		if err := validate(name); err != nil {
			resultErrors = append(resultErrors, errWrap(err))
			continue
		}

		if !slices.Contains(list, name) {
			resultErrors = append(resultErrors, errWrap(errs.ErrNotFound))
			continue
		}

		response, err := inspect(ctx, service, name)
		if err != nil {
			resultErrors = append(resultErrors, errWrap(err))
			continue
		}

		result = append(result, response)
	}

	return result, resultErrors
}

func Remove(ctx context.Context, cli *client.Client, names []string, removeCGroup bool) []error {
	service := cli.NamespaceService()
	resultErrors := []error{}

	for _, name := range names {
		err := remove(ctx, service, name, removeCGroup)
		if err != nil {
			resultErrors = append(resultErrors, errWrap(err))
			continue
		}
	}

	return resultErrors
}

func remove(ctx context.Context, service namespaces.Store, name string, removeCGroup bool) error {
	if err := validate(name); err != nil {
		return err
	}

	var deleteOptions []namespaces.DeleteOpts

	delOpt := getDeleteOptions(removeCGroup)
	if delOpt != nil {
		deleteOptions = append(deleteOptions, delOpt)
	}

	if err := service.Delete(NamespacedContext(ctx, name), name, deleteOptions...); err != nil {
		return errConvert(err)
	}

	return nil
}

func inspect(ctx context.Context, service namespaces.Store, name string) (*api.Namespace, error) {
	labels, err := service.Labels(NamespacedContext(ctx, name), name)
	if err != nil {
		return nil, errConvert(err)
	}

	return &api.Namespace{
		Name:   name,
		Labels: labels,
	}, nil
}

func update(ctx context.Context, service namespaces.Store, name string, key string, value string) error {
	err := service.SetLabel(NamespacedContext(ctx, name), name, key, value)
	if err != nil {
		return errConvert(err)
	}

	return nil
}

func errWrap(err error) error {
	return errors.Join(ErrServiceNamespace, err)
}

func errConvert(err error) error {
	if errdefs.IsNotFound(err) {
		return errors.Join(errs.ErrNotFound, err)
	}

	if errdefs.IsInvalidArgument(err) {
		return errors.Join(errs.ErrInvalidArgument, err)
	}

	if strings.Contains(err.Error(), "contains value with non-printable ASCII characters") {
		return errors.Join(errs.ErrInvalidArgument, err)
	}

	return errors.Join(errs.ErrSystemFailure, err)
}
