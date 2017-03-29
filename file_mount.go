package diecast

import (
	"mime"
	"net/http"
	"os"
	"path"
	"strings"
)

type FileMount struct {
	MountPoint  string          `json:"mount"`
	Path        string          `json:"source"`
	Passthrough bool            `json:"passthrough"`
	FileSystem  http.FileSystem `json:"-"`
}

func (self *FileMount) GetMountPoint() string {
	return self.MountPoint
}

func (self *FileMount) WillRespondTo(name string) bool {
	return strings.HasPrefix(name, self.GetMountPoint())
}

func (self *FileMount) OpenWithType(name string) (http.File, string, error) {
	if self.FileSystem == nil {
		if _, err := os.Stat(self.Path); err != nil {
			return nil, ``, err
		}
	}

	newPath := path.Join(strings.TrimSuffix(self.Path, `/`), strings.TrimPrefix(name, self.MountPoint))

	var file http.File
	var err error

	if self.FileSystem == nil {
		file, err = os.Open(newPath)
	} else {
		file, err = self.FileSystem.Open(newPath)
	}

	if err != nil {
		return nil, ``, err
	} else {
		return file, mime.TypeByExtension(path.Ext(newPath)), err
	}
}

func (self *FileMount) Open(name string) (http.File, error) {
	file, _, err := self.OpenWithType(name)
	return file, err
}
