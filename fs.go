// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/vfs source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package vfs

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var _ FileSystem = (*VFS)(nil)

// VFS errors
var (
	ErrMountExists    = errors.New("mount already exists")
	ErrMountNotExists = errors.New("mount does not exist")
)

// VFS represents Virtual File System (VFS), it operates in-memory.
// if file/directory doesn't exists on in-memory then it tries physical file system.
//
// VFS implements `vfs.FileSystem`, its a combination of package `os` and `ioutil`
// focused on Read-Only operations.
//
// Single point of access for all mounted virtual directories in aah application.
type VFS struct {
	mounts map[string]*Mount
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// VFS FileSystem interface methods
//______________________________________________________________________________

// Open method behaviour is same as `os.Open`.
func (v *VFS) Open(name string) (File, error) {
	m, err := v.FindMount(name)
	if err != nil {
		return nil, err
	}
	return m.Open(name)
}

// Lstat method behaviour is same as `os.Lstat`.
func (v *VFS) Lstat(name string) (os.FileInfo, error) {
	m, err := v.FindMount(name)
	if err != nil {
		return nil, err
	}
	return m.Lstat(name)
}

// Stat method behaviour is same as `os.Stat`
func (v *VFS) Stat(name string) (os.FileInfo, error) {
	m, err := v.FindMount(name)
	if err != nil {
		return nil, err
	}
	return m.Stat(name)
}

// ReadFile method behaviour is same as `ioutil.ReadFile`.
func (v *VFS) ReadFile(filename string) ([]byte, error) {
	m, err := v.FindMount(filename)
	if err != nil {
		return nil, err
	}
	return m.ReadFile(filename)
}

// ReadDir method behaviour is same as `ioutil.ReadDir`.
func (v *VFS) ReadDir(dirname string) ([]os.FileInfo, error) {
	m, err := v.FindMount(dirname)
	if err != nil {
		return nil, err
	}
	return m.ReadDir(dirname)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// VFS methods
//______________________________________________________________________________

// Walk method behaviour is same as `filepath.Walk`.
func (v *VFS) Walk(root string, walkFn filepath.WalkFunc) error {
	info, err := v.Lstat(root)
	if err == nil {
		err = walk(v, root, info, walkFn)
	} else {
		err = walkFn(root, nil, err)
	}

	if err == filepath.SkipDir {
		return nil
	}
	return err
}

// FindMount method finds the mounted virtual directory by mount path.
// if found then returns `Mount` instance otherwise nil and error.
//
// Mount implements `vfs.FileSystem`, its a combination of package `os` and `ioutil`
// focused on Read-Only operations.
func (v *VFS) FindMount(name string) (*Mount, error) {
	name = path.Clean(name)
	for _, m := range v.mounts {
		if m.vroot == name || strings.HasPrefix(name, m.tree.name+"/") {
			return m, nil
		}
	}
	return nil, &os.PathError{Op: "read", Path: name, Err: ErrMountNotExists}
}

// AddMount method used to mount physical directory as a virtual mounted directory.
//
// Basically aah scans and application source files and builds each file from
// mounted source dierctory into binary for single binary build.
func (v *VFS) AddMount(mountPath, physicalPath string) error {
	pp, err := filepath.Abs(filepath.Clean(physicalPath))
	if err != nil {
		return err
	}

	fi, err := os.Lstat(pp)
	if err != nil {
		return err
	}

	if !fi.IsDir() {
		return &os.PathError{Op: "addmount", Path: pp, Err: errors.New("is a file")}
	}

	mp := filepath.ToSlash(path.Clean(mountPath))
	if mp == "" {
		mp = path.Base(pp)
	}
	mp = path.Clean("/" + mp)

	if v.mounts == nil {
		v.mounts = make(map[string]*Mount)
	}

	if _, found := v.mounts[mp]; found {
		return &os.PathError{Op: "addmount", Path: mp, Err: ErrMountExists}
	}

	v.mounts[mp] = &Mount{
		vroot: mp,
		proot: pp,
		tree:  newNode(mp, fi),
	}

	return nil
}
