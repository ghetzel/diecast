package diecast

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/sliceutil"
)

// A FileMount exposes the contents of a given filesystem directory.
type FileMount struct {
	MountPoint      string                 `json:"mount"`
	Path            string                 `json:"source"`
	Passthrough     bool                   `json:"passthrough"`
	ResponseHeaders map[string]interface{} `json:"response_headers,omitempty"`
	ResponseCode    int                    `json:"response_code"`
	FileSystem      http.FileSystem        `json:"-"`
}

func (self *FileMount) GetMountPoint() string {
	return self.MountPoint
}

func (self *FileMount) GetTarget() string {
	return self.Path
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

	var newPath = path.Join(strings.TrimSuffix(self.Path, `/`), strings.TrimPrefix(name, self.MountPoint))

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
		var response = NewMountResponse(stat.Name(), stat.Size(), file)

		if mimetype, err := figureOutMimeType(newPath, file); err == nil {
			response.ContentType = mimetype
		} else {
			return nil, err
		}

		// add explicit response headers to response
		for name, value := range self.ResponseHeaders {
			value = strings.Join(sliceutil.Stringify(value), `,`)
			response.Metadata[name] = value
		}

		// override the response status code (if specified)
		if self.ResponseCode > 0 {
			response.StatusCode = self.ResponseCode
		} else if stat.IsDir() {
			if strings.HasSuffix(req.URL.Path, `/`) {
				return response, fmt.Errorf("is a directory")
			} else {
				response.RedirectCode = http.StatusMovedPermanently
			}
		}

		return response, nil

	} else {
		return nil, err
	}
}

func (self *FileMount) String() string {
	return fmt.Sprintf("%T('%s')", self, self.GetMountPoint())
}

func (self *FileMount) Open(name string) (http.File, error) {
	return openAsHttpFile(self, name)
}

func figureOutMimeType(filename string, file io.ReadSeeker) (string, error) {
	var mimetype string

	if mimetype = fileutil.GetMimeType(path.Ext(filename)); mimetype != `` {
		return mimetype, nil
	}

	if file != nil {
		defer file.Seek(0, io.SeekStart)

		if mimetype = fileutil.GetMimeType(file); mimetype != `` {
			return mimetype, nil
		}
	}

	return ``, nil
}
