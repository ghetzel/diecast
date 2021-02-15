package diecast

import (
	"container/list"
	"net/http"
)

// Implements a simple, pluggable Virtual File System
type VFS struct {
	filesystems list.List
	fallback    http.FileSystem
	overrides   map[string]interface{}
}

// Append the given filesystem to the end, making it the lowest priority for lookups.
func (self *VFS) AppendFS(fs http.FileSystem) {
	if fs != nil {
		self.filesystems.PushBack(fs)
	}
}

// Prepend the given filesystem to the end, making it the highest priority for lookups.
func (self *VFS) PrependFS(fs http.FileSystem) {
	if fs != nil {
		self.filesystems.PushFront(fs)
	}
}

// Explicitly provide the desired response to retrieving a specific path.  The data can be a variety
// of types, including http.File, string, []byte, io.Reader, and error.
func (self *VFS) OverridePath(name string, data interface{}) {
	if len(self.overrides) == 0 {
		self.overrides = make(map[string]interface{})
	}

	self.overrides[name] = data
}

// Set the filesystem that will be used to respond to any requests not otherwise handled by plugins and overrides.
func (self *VFS) SetFallbackFS(fallback http.FileSystem) {
	if fallback != nil {
		self.fallback = fallback
	}
}

// Retrieve a file from the VFS.
func (self *VFS) Open(name string) (http.File, error) {
	if len(self.overrides) > 0 {
		if v, ok := self.overrides[name]; ok {
			return newMockHttpFile(name, v)
		}
	}

	// search plugged in filesystems next
	for el := self.filesystems.Front(); el != nil; el = el.Next() {
		if fs, ok := el.Value.(http.FileSystem); ok {
			if file, err := fs.Open(name); err == nil {
				return file, nil
			} else if _, ok := err.(*ControlError); ok {
				return file, err
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
