package diecast

import (
	"net/http"
	"os"
	"path"
	"strings"
)

type Mount struct {
	http.FileSystem
	MountPoint  string `json:"mount"`
	Path        string `json:"path"`
	Passthrough bool   `json:"passthrough"`
}

func NewMountFromSpec(spec string) (*Mount, error) {
	parts := strings.SplitN(spec, `:`, 2)
	var path string
	var mountPoint string

	if len(parts) == 1 {
		path = parts[0]
		mountPoint = parts[0]
	} else {
		path = parts[0]
		mountPoint = parts[1]
	}

	mount := &Mount{
		Path:       path,
		MountPoint: mountPoint,
	}

	if err := mount.Initialize(); err != nil {
		return nil, err
	}

	return mount, nil
}

func (self *Mount) Initialize() error {
	if _, err := os.Stat(self.Path); err != nil {
		return err
	}

	log.Debugf("Initialize mount %q -> %q", self.MountPoint, self.Path)

	return nil
}

func (self *Mount) WillRespondTo(name string) bool {
	return strings.HasPrefix(name, self.MountPoint)
}

func (self *Mount) Open(name string) (http.File, error) {
	newPath := path.Join(strings.TrimSuffix(self.Path, `/`), strings.TrimPrefix(name, self.MountPoint))
	return os.Open(newPath)
}
