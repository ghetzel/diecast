package diecast

import (
	"net/http"

	"github.com/ghetzel/go-stockutil/typeutil"
)

type File struct {
	Path   string      `yaml:"path"`
	Source string      `yaml:"source"`
	Data   interface{} `yaml:"data"`
}

// returns an object that satisfies the http.File interface and returns data as read from the Source
// file path, or from the data literal.  If Data is a map or array, it will be encoded and returned as an encoded string.
func (self *File) httpFile(vfs *VFS) (http.File, error) {
	if self.Data != nil {
		var mimetype string
		var data = self.Data

		if typeutil.IsMap(data) || typeutil.IsArray(data) {
			if b, m, err := AutoencodeByFilename(self.Path, data); err == nil {
				data = b
				mimetype = m
			} else {
				return nil, err
			}
		}

		if file, err := newMockHttpFile(self.Path, data); err == nil {
			if mimetype != `` {
				file.SetHeader(`Content-Type`, mimetype)
			}

			return file, nil
		} else {
			return nil, err
		}
	} else if vfs != nil && self.Source != `` {
		return vfs.Open(self.Source)
	} else {
		return nil, ErrNotFound
	}
}
