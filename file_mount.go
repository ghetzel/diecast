package diecast

import (
	"fmt"
	"io"
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

func (self *FileMount) WillRespondTo(name string, req *http.Request, requestBody io.Reader) bool {
	return strings.HasPrefix(name, self.GetMountPoint())
}

func (self *FileMount) OpenWithType(name string, req *http.Request, requestBody io.Reader) (*MountResponse, error) {
	if self.FileSystem == nil {
		if _, err := os.Stat(self.Path); err != nil {
			return nil, err
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
		return nil, err
	} else if file == nil {
		return nil, fmt.Errorf("Invalid file object for '%v'", name)
	} else if stat, err := file.Stat(); err == nil {
		response := NewMountResponse(stat.Name(), stat.Size(), file)
		response.ContentType = mime.TypeByExtension(path.Ext(newPath))

		if stat.IsDir() {
			response.RedirectCode = http.StatusMovedPermanently
		}

		return response, nil

	} else {
		return nil, err
	}
}

func (self *FileMount) Open(name string) (http.File, error) {
	return openAsHttpFile(self, name)
}
