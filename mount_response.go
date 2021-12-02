package diecast

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type MountResponse struct {
	ContentType        string
	StatusCode         int
	Metadata           map[string]interface{}
	RedirectTo         string
	RedirectCode       int
	payload            interface{}
	name               string
	size               int64
	underlyingFile     http.File
	underlyingFileInfo os.FileInfo
}

func NewMountResponse(name string, size int64, payload interface{}) *MountResponse {
	return &MountResponse{
		ContentType: `application/octet-stream`,
		StatusCode:  http.StatusOK,
		Metadata:    make(map[string]interface{}),
		payload:     payload,
		name:        name,
		size:        size,
	}
}

func (self *MountResponse) setUnderlyingFile(file http.File, info os.FileInfo) {
	self.underlyingFile = file
	self.underlyingFileInfo = info
}

func (self *MountResponse) GetPayload() interface{} {
	return self.payload
}

func (self *MountResponse) GetFile() http.File {
	if file, ok := self.payload.(http.File); ok {
		return file
	} else {
		return self
	}
}

func (self *MountResponse) Read(p []byte) (int, error) {
	if self.payload == nil {
		return 0, fmt.Errorf("Cannot read from closed response")
	} else if reader, ok := self.payload.(io.Reader); ok {
		return reader.Read(p)
	} else {
		return 0, fmt.Errorf("Payload does not implement io.ReadSeeker")
	}
}

func (self *MountResponse) Seek(offset int64, whence int) (int64, error) {
	if seeker, ok := self.payload.(io.Seeker); ok {
		return seeker.Seek(offset, whence)
	} else {
		return 0, fmt.Errorf("Payload is not seekable")
	}
}

func (self *MountResponse) Close() error {
	if closer, ok := self.payload.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			return err
		}
	}

	self.payload = nil
	return nil
}

func (self *MountResponse) Readdir(count int) ([]os.FileInfo, error) {
	if self.underlyingFileInfo != nil {
		return self.underlyingFile.Readdir(count)
	} else {
		return nil, fmt.Errorf("readdir() not available on this response object")
	}
}

func (self *MountResponse) Name() string {
	if self.underlyingFileInfo != nil {
		return self.underlyingFileInfo.Name()
	} else {
		return self.name
	}
}

func (self *MountResponse) Size() int64 {
	if self.underlyingFileInfo != nil {
		return self.underlyingFileInfo.Size()
	} else {
		return self.size
	}
}

func (self *MountResponse) Mode() os.FileMode {
	if self.underlyingFileInfo != nil {
		return self.underlyingFileInfo.Mode()
	} else {
		return 0666
	}
}

func (self *MountResponse) ModTime() time.Time {
	if self.underlyingFileInfo != nil {
		return self.underlyingFileInfo.ModTime()
	} else {
		return time.Now()
	}
}

func (self *MountResponse) IsDir() bool {
	if self.underlyingFileInfo != nil {
		return self.underlyingFileInfo.IsDir()
	} else {
		return false
	}
}

func (self *MountResponse) Sys() interface{} {
	if self.underlyingFileInfo != nil {
		return self.underlyingFileInfo.Sys()
	} else {
		return nil
	}
}

func (self *MountResponse) Stat() (os.FileInfo, error) {
	return self, nil
}
