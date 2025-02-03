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

package store

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"go.farcloser.world/core/filesystem"

	"go.farcloser.world/lepton/leptonic/errs"
)

const (
	// Default filesystem permissions to use when creating dir or files
	defaultFilePerm    = 0o600
	defaultDirPerm     = 0o700
	transformBlockSize = 64 // grouping of chars per directory depth
)

func transform(keys ...string) []string {
	key := fmt.Sprintf("%x", sha256.Sum256([]byte(filepath.Join(keys...))))

	var (
		sliceSize = len(key) / transformBlockSize
		pathSlice = make([]string, sliceSize)
	)

	for i := range sliceSize {
		from, to := i*transformBlockSize, (i+1)*transformBlockSize
		pathSlice[i] = key[from:to]
	}

	return pathSlice
}

// New returns a filesystem based Store implementation that satisfies both Manager and Locker
// Note that atomicity is "guaranteed" by `os.Rename`, which arguably is not *always* atomic.
// In particular, operating-system crashes may break that promise, and windows behavior is probably questionable.
// That being said, this is still a much better solution than writing directly to the destination file.
func New(rootPath string, hashPath bool, dirPerm os.FileMode, filePerm os.FileMode) (Store, error) {
	if rootPath == "" {
		return nil, errors.Join(errs.ErrInvalidArgument, errors.New("FileStore rootPath cannot be empty"))
	}

	if dirPerm == 0 {
		dirPerm = defaultDirPerm
	}

	if filePerm == 0 {
		filePerm = defaultFilePerm
	}

	if err := os.MkdirAll(rootPath, dirPerm); err != nil {
		return nil, errors.Join(errs.ErrSystemFailure, err)
	}

	fs := &fileStore{
		dir:      rootPath,
		dirPerm:  dirPerm,
		filePerm: filePerm,
	}

	if hashPath {
		fs.transform = transform
	}

	return fs, nil
}

type fileStore struct {
	mutex     sync.RWMutex
	dir       string
	locked    *os.File
	dirPerm   os.FileMode
	filePerm  os.FileMode
	transform func(...string) []string
}

func (vs *fileStore) Lock() error {
	vs.mutex.Lock()

	dirFile, err := filesystem.Lock(vs.dir)
	if err != nil {
		return err
	}

	vs.locked = dirFile

	return nil
}

func (vs *fileStore) ReadOnlyLock() error {
	vs.mutex.Lock()

	dirFile, err := filesystem.ReadOnlyLock(vs.dir)
	if err != nil {
		return err
	}

	vs.locked = dirFile

	return nil
}

func (vs *fileStore) Release() error {
	defer vs.mutex.Unlock()

	defer func() {
		vs.locked = nil
	}()

	if err := filesystem.Unlock(vs.locked); err != nil {
		if errors.Is(err, filesystem.ErrLockIsNil) {
			return errors.Join(errs.ErrFaultyImplementation, fmt.Errorf("cannot unlock already unlocked volume store %q", vs.dir))
		}

		return err
	}

	return nil
}

func (vs *fileStore) WithReadOnlyLock(fun func() error) (err error) {
	if err = vs.ReadOnlyLock(); err != nil {
		return err
	}

	defer func() {
		err = errors.Join(vs.Release(), err)
	}()

	return fun()
}

func (vs *fileStore) WithLock(fun func() error) (err error) {
	if err = vs.Lock(); err != nil {
		return err
	}

	defer func() {
		err = errors.Join(vs.Release(), err)
	}()

	return fun()
}

func (vs *fileStore) Get(key ...string) ([]byte, error) {
	if vs.locked == nil {
		return nil, errors.Join(errs.ErrFaultyImplementation, errors.New("operations on the store must use locking"))
	}

	path, err := vs.Location(key...)
	if err != nil {
		return nil, err
	}

	st, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errors.Join(errs.ErrNotFound, fmt.Errorf("%q does not exist", filepath.Join(key...)))
		}

		return nil, errors.Join(errs.ErrSystemFailure, err)
	}

	if st.IsDir() {
		return nil, errors.Join(errs.ErrFaultyImplementation, fmt.Errorf("%q is a directory and cannot be read as a file", path))
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Join(errs.ErrSystemFailure, err)
	}

	return content, nil
}

func (vs *fileStore) Exists(key ...string) (bool, error) {
	path, err := vs.Location(key...)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}

		return false, errors.Join(errs.ErrSystemFailure, err)
	}

	return true, nil
}

func (vs *fileStore) Set(data []byte, key ...string) error {
	if vs.locked == nil {
		return errors.Join(errs.ErrFaultyImplementation, errors.New("operations on the store must use locking"))
	}

	path, err := vs.Location(key...)
	if err != nil {
		return err
	}

	parent := filepath.Dir(path)

	if parent != vs.dir {
		err := os.MkdirAll(parent, vs.dirPerm)
		if err != nil {
			return errors.Join(errs.ErrSystemFailure, err)
		}
	}

	st, err := os.Stat(path)
	if err == nil {
		if st.IsDir() {
			return errors.Join(errs.ErrFaultyImplementation, fmt.Errorf("%q is a directory and cannot be written to", path))
		}
	}

	return filesystem.WriteFile(path, data, vs.filePerm)
}

func (vs *fileStore) List(key ...string) ([]string, error) {
	// NOTE: list is problematic when hashing, as the returned keys may not make sense
	if vs.locked == nil {
		return nil, errors.Join(errs.ErrFaultyImplementation, errors.New("operations on the store must use locking"))
	}

	var err error
	path := vs.dir

	// Unlike Get, Set and Delete, List can have zero length key
	if len(key) > 0 {
		path, err = vs.Location(key...)
		if err != nil {
			return nil, err
		}
	}

	st, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errors.Join(errs.ErrNotFound, err)
		}

		return nil, errors.Join(errs.ErrSystemFailure, err)
	}

	if !st.IsDir() {
		return nil, errors.Join(errs.ErrFaultyImplementation, fmt.Errorf("%q is not a directory and cannot be enumerated", path))
	}

	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, errors.Join(errs.ErrSystemFailure, err)
	}

	entries := []string{}
	for _, dirEntry := range dirEntries {
		entries = append(entries, dirEntry.Name())
	}

	return entries, nil
}

func (vs *fileStore) Delete(key ...string) error {
	if vs.locked == nil {
		return errors.Join(errs.ErrFaultyImplementation, errors.New("operations on the store must use locking"))
	}

	path, err := vs.Location(key...)
	if err != nil {
		return err
	}

	_, err = os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errors.Join(errs.ErrNotFound, err)
		}

		return errors.Join(errs.ErrSystemFailure, err)
	}

	if err = os.RemoveAll(path); err != nil {
		return errors.Join(errs.ErrSystemFailure, err)
	}

	return nil
}

func (vs *fileStore) Location(key ...string) (string, error) {
	if vs.transform != nil {
		key = vs.transform(key...)
	}

	if err := validateAllPathComponents(key...); err != nil {
		return "", err
	}

	return filepath.Join(vs.dir, filepath.Join(key...)), nil
}

func (vs *fileStore) GroupEnsure(key ...string) error {
	if vs.locked == nil {
		return errors.Join(errs.ErrFaultyImplementation, errors.New("operations on the store must use locking"))
	}

	path, err := vs.Location(key...)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(path, vs.dirPerm); err != nil {
		return errors.Join(errs.ErrSystemFailure, err)
	}

	return nil
}

func (vs *fileStore) GroupSize(key ...string) (int64, error) {
	if vs.locked == nil {
		return 0, errors.Join(errs.ErrFaultyImplementation, errors.New("operations on the store must use locking"))
	}

	path, err := vs.Location(key...)
	if err != nil {
		return 0, err
	}

	st, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, errors.Join(errs.ErrNotFound, err)
		}

		return 0, errors.Join(errs.ErrSystemFailure, err)
	}

	if !st.IsDir() {
		return 0, errors.Join(errs.ErrFaultyImplementation, fmt.Errorf("%q is not a directory", path))
	}

	var size int64
	var walkFn = func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	}

	err = filepath.Walk(path, walkFn)
	if err != nil {
		return 0, err
	}

	return size, nil
}

// validateAllPathComponents will enforce validation for a slice of components
func validateAllPathComponents(pathComponent ...string) error {
	if len(pathComponent) == 0 {
		return errors.Join(errs.ErrInvalidArgument, errors.New("you must specify an identifier"))
	}

	for _, key := range pathComponent {
		if err := filesystem.ValidatePathComponent(key); err != nil {
			return errors.Join(errs.ErrInvalidArgument, err)
		}
	}

	return nil
}
