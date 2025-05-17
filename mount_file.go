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
	MountPoint      string          `json:"mount"`
	Path            string          `json:"source"`
	Passthrough     bool            `json:"passthrough"`
	ResponseHeaders map[string]any  `json:"response_headers,omitempty"`
	ResponseCode    int             `json:"response_code"`
	FileSystem      http.FileSystem `json:"-"`
}

func (mount *FileMount) GetMountPoint() string {
	return mount.MountPoint
}

func (mount *FileMount) GetTarget() string {
	return mount.Path
}

func (mount *FileMount) WillRespondTo(name string, req *http.Request, requestBody io.Reader) bool {
	return strings.HasPrefix(name, mount.GetMountPoint())
}

func (mount *FileMount) OpenWithType(name string, req *http.Request, requestBody io.Reader) (*MountResponse, error) {
	if mount.FileSystem == nil {
		if _, err := os.Stat(mount.Path); err != nil {
			return nil, err
		}
	}

	var newPath = path.Join(strings.TrimSuffix(mount.Path, `/`), strings.TrimPrefix(name, mount.MountPoint))

	var file http.File
	var err error

	if mount.FileSystem == nil {
		file, err = os.Open(newPath)
	} else {
		file, err = mount.FileSystem.Open(newPath)
	}

	if err != nil {
		return nil, err
	} else if file == nil {
		return nil, fmt.Errorf("invalid file object for '%v'", name)
	} else if stat, err := file.Stat(); err == nil {
		var response = NewMountResponse(stat.Name(), stat.Size(), file)

		if mimetype, err := figureOutMimeType(newPath, file); err == nil {
			response.ContentType = mimetype
		} else {
			return nil, err
		}

		// add explicit response headers to response
		for name, value := range mount.ResponseHeaders {
			value = strings.Join(sliceutil.Stringify(value), `,`)
			response.Metadata[name] = value
		}

		// override the response status code (if specified)
		if mount.ResponseCode > 0 {
			response.StatusCode = mount.ResponseCode
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

func (mount *FileMount) String() string {
	return fmt.Sprintf("%T('%s')", mount, mount.GetMountPoint())
}

func (mount *FileMount) Open(name string) (http.File, error) {
	return openAsHttpFile(mount, name)
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
