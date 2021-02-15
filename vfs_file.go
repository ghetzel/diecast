package diecast

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ghetzel/go-stockutil/typeutil"
	"gopkg.in/yaml.v3"
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
		var data = self.Data

		if typeutil.IsMap(data) || typeutil.IsArray(data) {
			if b, err := json.MarshalIndent(data, ``, `  `); err == nil {
				data = b
			} else {
				return nil, err
			}
		}

		return newMockHttpFile(self.Path, data)
	} else if vfs != nil && self.Source != `` {
		return vfs.Open(self.Source)
	} else {
		return nil, ErrNotFound
	}
}

func (self *File) aetype() string {
	var ext string

	ext = filepath.Base(self.Path)
	ext = filepath.Ext(ext)
	ext = strings.ToLower(ext)

	return ext
}

func (self *File) autoencode(data interface{}) ([]byte, error) {
	switch self.aetype() {
	case `yaml`, `yml`:
		return yaml.Marshal(data)
	case `json`, ``:
		return json.MarshalIndent(data, ``, `  `)
	default:
		return typeutil.Bytes(data), nil
	}
}
