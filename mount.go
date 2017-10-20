package diecast

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

var MountHaltErr = errors.New(`mount halted`)

type Mount interface {
	Open(string) (http.File, error)
	OpenWithType(string, *http.Request, io.Reader) (*MountResponse, error)
	WillRespondTo(string, *http.Request, io.Reader) bool
	GetMountPoint() string
	String() string
}

func NewMountFromSpec(spec string) (Mount, error) {
	parts := strings.SplitN(spec, `:`, 2)
	var mountPoint string
	var source string
	var scheme string

	if len(parts) == 1 {
		mountPoint = parts[0]
		source = parts[0]
	} else {
		mountPoint = parts[0]
		source = parts[1]
	}

	sourceParts := strings.SplitN(source, `:`, 2)

	if len(sourceParts) == 2 {
		scheme = sourceParts[0]
	}

	var mount Mount

	switch scheme {
	case `http`, `https`:
		mount = &ProxyMount{
			URL:        source,
			MountPoint: mountPoint,
		}

	default:
		if absPath, err := filepath.Abs(source); err == nil {
			source = absPath
		} else {
			return nil, err
		}

		mount = &FileMount{
			Path:       source,
			MountPoint: mountPoint,
		}
	}

	log.Debugf("Creating mount %T: %+v", mount, mount)

	return mount, nil
}

func IsHardStop(err error) bool {
	if err != nil && err.Error() == `mount halted` {
		return true
	}

	return false
}

func IsDirectoryError(err error) bool {
	if err != nil && err.Error() == `is a directory` {
		return true
	}

	return false
}

func openAsHttpFile(mount Mount, name string) (http.File, error) {
	if file, err := mount.OpenWithType(name, nil, nil); err == nil {
		if hfile, ok := file.GetPayload().(http.File); ok && hfile != nil {
			return hfile, nil
		} else {
			return nil, fmt.Errorf("Wrong response type")
		}
	} else {
		return nil, err
	}
}
