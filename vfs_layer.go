package diecast

import (
	"fmt"
	"io/fs"

	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

// A layer represents a filesystem from which files can be retrieved and read.
type Layer struct {
	Type          string                 `yaml:"type"`
	Options       map[string]interface{} `yaml:"options"`
	Paths         interface{}            `yaml:"paths"`
	HaltOnMissing bool                   `yaml:"haltOnMissing"`
	HaltOnError   bool                   `yaml:"haltOnError"`
	fs            fs.FS
}

// Return a typeutil.Variant containing the value at the named option key, or a fallback value.
func (self *Layer) Option(name string, fallbacks ...interface{}) typeutil.Variant {
	return maputil.M(self.Options).Get(name, fallbacks...)
}

// Return whether this layer is configured to respond to requests for the given filename.
func (self *Layer) shouldConsiderOpening(name string) bool {
	var validPatterns = sliceutil.Stringify(self.Paths)

	if len(validPatterns) == 0 {
		return true
	} else {
		for _, pattern := range validPatterns {
			if IsGlobMatch(name, pattern) {
				return true
			}
		}
	}

	return false
}

// Retrieve the named file from the filesystem specified
func (self *Layer) openFsFile(name string) (fs.File, error) {
	if self.fs == nil {
		if fsfn, ok := filesystems[self.Type]; ok {
			if fsfn != nil {
				if fs, err := fsfn(self); err == nil {
					self.fs = fs
				} else {
					return nil, fmt.Errorf("layer filesystem: %v", err)
				}
			}
		}
	}

	if self.fs != nil {
		return self.fs.Open(name)
	} else {
		return nil, fmt.Errorf("invalid layer")
	}
}
