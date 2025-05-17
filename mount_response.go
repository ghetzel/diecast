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
	Metadata           map[string]any
	RedirectTo         string
	RedirectCode       int
	payload            any
	name               string
	size               int64
	underlyingFile     http.File
	underlyingFileInfo os.FileInfo
}

func NewMountResponse(name string, size int64, payload any) *MountResponse {
	return &MountResponse{
		ContentType: `application/octet-stream`,
		StatusCode:  http.StatusOK,
		Metadata:    make(map[string]any),
		payload:     payload,
		name:        name,
		size:        size,
	}
}

func (response *MountResponse) setUnderlyingFile(file http.File, info os.FileInfo) {
	response.underlyingFile = file
	response.underlyingFileInfo = info
}

func (response *MountResponse) GetPayload() any {
	return response.payload
}

func (response *MountResponse) GetFile() http.File {
	if file, ok := response.payload.(http.File); ok {
		return file
	} else {
		return response
	}
}

func (response *MountResponse) Read(p []byte) (int, error) {
	if response.payload == nil {
		return 0, fmt.Errorf("cannot read from closed response")
	} else if reader, ok := response.payload.(io.Reader); ok {
		return reader.Read(p)
	} else {
		return 0, fmt.Errorf("payload does not implement io.ReadSeeker")
	}
}

func (response *MountResponse) Seek(offset int64, whence int) (int64, error) {
	if seeker, ok := response.payload.(io.Seeker); ok {
		return seeker.Seek(offset, whence)
	} else {
		return 0, fmt.Errorf("payload is not seekable")
	}
}

func (response *MountResponse) Close() error {
	if closer, ok := response.payload.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			return err
		}
	}

	response.payload = nil
	return nil
}

func (response *MountResponse) Readdir(count int) ([]os.FileInfo, error) {
	if response.underlyingFileInfo != nil {
		return response.underlyingFile.Readdir(count)
	} else {
		return nil, fmt.Errorf("readdir() not available on this response object")
	}
}

func (response *MountResponse) Name() string {
	if response.underlyingFileInfo != nil {
		return response.underlyingFileInfo.Name()
	} else {
		return response.name
	}
}

func (response *MountResponse) Size() int64 {
	if response.underlyingFileInfo != nil {
		return response.underlyingFileInfo.Size()
	} else {
		return response.size
	}
}

func (response *MountResponse) Mode() os.FileMode {
	if response.underlyingFileInfo != nil {
		return response.underlyingFileInfo.Mode()
	} else {
		return 0666
	}
}

func (response *MountResponse) ModTime() time.Time {
	if response.underlyingFileInfo != nil {
		return response.underlyingFileInfo.ModTime()
	} else {
		return time.Now()
	}
}

func (response *MountResponse) IsDir() bool {
	if response.underlyingFileInfo != nil {
		return response.underlyingFileInfo.IsDir()
	} else {
		return false
	}
}

func (response *MountResponse) Sys() any {
	if response.underlyingFileInfo != nil {
		return response.underlyingFileInfo.Sys()
	} else {
		return nil
	}
}

func (response *MountResponse) Stat() (os.FileInfo, error) {
	return response, nil
}
