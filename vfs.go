package diecast

import (
	"io/fs"
	"strings"
)

type FileSystemFunc = func(*Layer) (fs.FS, error)

// Register a new filesystem creator function to the given type.  If type is empty,
// the given function will be used as the default filesystem for unspecified layer types.
func RegisterFS(fstype string, fsfn FileSystemFunc) {
	filesystems[fstype] = fsfn
}

// Implements a simple, pluggable Virtual File System
type VFS struct {
	Layers    []Layer          `yaml:"layers"`
	overrides map[string]*File `yaml:"overrides"`
	fallback  fs.FS
}

// Set the filesystem that will be used to respond to any requests not otherwise handled by plugins and overrides.
func (self *VFS) SetFallbackFS(fallback fs.FS) {
	if fallback != nil {
		self.fallback = fallback
	}
}

// Override a given path to explicitly return specified data.
func (self *VFS) AddOverride(path string, data any) {
	if len(self.overrides) == 0 {
		self.overrides = make(map[string]*File)
	}

	// normalize pathname to FS root
	path = `/` + strings.TrimPrefix(path, `/`)

	self.overrides[path] = &File{
		Path: path,
		Data: data,
	}
}

// Retrieve a file from the VFS.
func (self *VFS) Open(name string) (fs.File, error) {
	// normalize pathname to FS root
	name = `/` + strings.TrimPrefix(name, `/`)

	// check for explicitly overridden filenames and return them if present
	if ov, ok := self.overrides[name]; ok {
		return ov.fsFile(self)
	}

	// search through layers to find one that matches
	for _, layer := range self.Layers {
		if layer.shouldConsiderOpening(name) {
			if file, err := layer.openFsFile(name); err == nil {
				return file, nil
			} else if err == ErrNotFound {
				if layer.HaltOnMissing {
					return nil, err
				}
			} else if layer.HaltOnError {
				return nil, err
			} else {
				continue
			}
		}
	}

	// search fallback fs, and respond with Not Found as a last resort
	if fs := self.fallback; fs != nil {
		if file, err := fs.Open(name); err == nil {
			if stat, err := file.Stat(); err == nil {
				if !stat.IsDir() {
					return file, nil
				}
			}
		}
	}

	return nil, ErrNotFound
}
