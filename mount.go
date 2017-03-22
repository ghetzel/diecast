package diecast

import (
	"net/http"
	"os"
	"path"
	"strings"
)

type Mount struct {
	MountPoint  string          `json:"mount"`
	Path        string          `json:"path"`
	Passthrough bool            `json:"passthrough"`
	FileSystem  http.FileSystem `json:"-"`
}

func NewMountFromSpec(spec string) (*Mount, error) {
	parts := strings.SplitN(spec, `:`, 2)
	var fsPath string
	var mountPoint string

	if len(parts) == 1 {
		fsPath = parts[0]
		mountPoint = parts[0]
	} else {
		fsPath = parts[0]
		mountPoint = parts[1]
	}

	if !strings.HasPrefix(fsPath, `/`) {
		if cwd, err := os.Getwd(); err == nil {
			fsPath = path.Join(cwd, fsPath)
		} else {
			return nil, err
		}
	}

	mount := &Mount{
		Path:       fsPath,
		MountPoint: mountPoint,
	}

	if err := mount.Initialize(); err != nil {
		return nil, err
	}

	return mount, nil
}

func (self *Mount) Initialize() error {
	if self.FileSystem == nil {
		if _, err := os.Stat(self.Path); err != nil {
			return err
		}
	}

	log.Debugf("Initialize mount %q -> %q", self.MountPoint, self.Path)

	return nil
}

func (self *Mount) WillRespondTo(name string) bool {
	return strings.HasPrefix(name, self.MountPoint)
}

func (self *Mount) OpenFile(name string) (http.File, error) {
	newPath := path.Join(strings.TrimSuffix(self.Path, `/`), strings.TrimPrefix(name, self.MountPoint))

	log.Debugf("OpenFile(%q)", newPath)

	if self.FileSystem == nil {
		return os.Open(newPath)
	} else {
		return self.FileSystem.Open(newPath)
	}
}

func (self *Mount) Open(name string) (http.File, error) {
	return self.OpenFile(name)
}
