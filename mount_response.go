package diecast

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type MountResponse struct {
	ContentType  string
	Metadata     map[string]interface{}
	RedirectTo   string
	RedirectCode int
	payload      interface{}
	name         string
	size         int64
}

func NewMountResponse(name string, size int64, payload interface{}) *MountResponse {
	return &MountResponse{
		ContentType: `application/octet-stream`,
		Metadata:    make(map[string]interface{}),
		payload:     payload,
		name:        name,
		size:        size,
	}
}

func (self *MountResponse) GetPayload() interface{} {
	return self.payload
}

func (self *MountResponse) GetFile() http.File {
	if file, ok := self.payload.(http.File); ok {
		return file
	} else {
		return nil
	}
}

func (self *MountResponse) Read(p []byte) (int, error) {
	if self.payload == nil {
		return 0, fmt.Errorf("Cannot read from closed response")
	} else if reader, ok := self.payload.(io.ReadSeeker); ok {
		return reader.Read(p)
	} else {
		return 0, fmt.Errorf("Payload does not implement io.ReadSeeker")
	}
}

func (self *MountResponse) Seek(offset int64, whence int) (int64, error) {
	if readSeeker, ok := self.payload.(io.ReadSeeker); ok {
		return readSeeker.Seek(offset, whence)
	} else {
		return 0, fmt.Errorf("Payload is not seekable")
	}
}

func (self *MountResponse) Close() error {
	self.payload = nil
	return nil
}

func (self *MountResponse) Readdir(count int) ([]os.FileInfo, error) {
	return nil, fmt.Errorf("readdir() not valid on response objects")
}

func (self *MountResponse) Name() string {
	return self.name
}

func (self *MountResponse) Size() int64 {
	return self.size
}

func (self *MountResponse) Mode() os.FileMode {
	return 0666
}

func (self *MountResponse) ModTime() time.Time {
	return time.Now()
}

func (self *MountResponse) IsDir() bool {
	return false
}

func (self *MountResponse) Sys() interface{} {
	return nil
}

func (self *MountResponse) Stat() (os.FileInfo, error) {
	return self, nil
}
