package diecast

import (
	"io/fs"
)

type FileSystemFunc = func(*Layer) (fs.FS, error)

// Register a new filesystem creator function to the given type.  If type is empty,
// the given function will be used as the default filesystem for unspecified layer types.
func RegisterFS(fstype string, fsfn FileSystemFunc) {
	filesystems[fstype] = fsfn
}

// Implements a simple, pluggable Virtual File System
type VFS struct {
	Overrides map[string]*File `yaml:"overrides"`
	Layers    []Layer          `yaml:"layers"`
	fallback  fs.FS
}

// Set the filesystem that will be used to respond to any requests not otherwise handled by plugins and overrides.
func (self *VFS) SetFallbackFS(fallback fs.FS) {
	if fallback != nil {
		self.fallback = fallback
	}
}

// Retrieve a file from the VFS.
func (self *VFS) Open(name string) (fs.File, error) {
	if ov, ok := self.Overrides[name]; ok {
		// log.Debugf("vfs: open %s [override]", name)
		return ov.fsFile(self)
	}

	// search through layers
	for _, layer := range self.Layers {
		if layer.shouldConsiderOpening(name) {
			if file, err := layer.openFsFile(name); err == nil {
				// log.Debugf("vfs: open %s [layer=%d]", name, i)
				return file, nil
			} else if err == ErrNotFound {
				if layer.HaltOnMissing {
					// log.Debugf("vfs: halt: missing %s [layer=%d]", name, i)
					return nil, err
				}
			} else if layer.HaltOnError {
				// log.Debugf("vfs: halt: error %v [layer=%d]", err, i)
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
					// log.Debugf("vfs: open %s [fallback]", name)
					return file, nil
				}
			}
		}
	}

	return nil, ErrNotFound
}
