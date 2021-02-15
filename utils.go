package diecast

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/gobwas/glob"
	"gopkg.in/yaml.v2"
)

var globcache sync.Map

// Return whether the give path matches the given extended glob pattern as described
// at https://pkg.go.dev/github.com/gobwas/glob#Compile
//
func IsGlobMatch(path string, pattern string) bool {
	var globber glob.Glob

	if v, ok := globcache.Load(pattern); ok && v != nil {
		globber = v.(glob.Glob)
	} else if g, err := glob.Compile(pattern); err == nil {
		globber = g
		globcache.Store(pattern, g)
	} else {
		return false
	}

	return globber.Match(path)
}

func AutoencodeByFilename(name string, data interface{}) ([]byte, string, error) {
	var ext string

	ext = filepath.Base(name)
	ext = filepath.Ext(ext)
	ext = strings.ToLower(ext)

	var mimetype = fileutil.GetMimeType(name)
	var b []byte
	var err error

	switch ext {
	case `.yaml`, `.yml`:
		b, err = yaml.Marshal(data)
	default:
		if typeutil.IsMap(data) || typeutil.IsArray(data) {
			b, err = json.MarshalIndent(data, ``, `  `)
			mimetype = `application/json`
		} else {
			b = typeutil.Bytes(data)
		}
	}

	if mimetype == `` {
		mimetype = `application/octet-stream`
	}

	return b, mimetype, err
}
