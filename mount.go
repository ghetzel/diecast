package diecast

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/stringutil"
)

type MountConfig struct {
	Mount   string                 `yaml:"mount"   json:"mount"`   // The URL path that this mount will respond to
	To      string                 `yaml:"to"      json:"to"`      // The upstream URL or local filesystem path that will serve this path
	Options map[string]interface{} `yaml:"options" json:"options"` // Mount-specific options
}

var MountHaltErr = errors.New(`mount halted`)

type Mount interface {
	Open(string) (http.File, error)
	OpenWithType(string, *http.Request, io.Reader) (*MountResponse, error)
	WillRespondTo(string, *http.Request, io.Reader) bool
	GetMountPoint() string
	GetTarget() string
}

func NewMountFromSpec(spec string) (Mount, error) {
	mountPoint, source := stringutil.SplitPair(spec, `:`)

	if source == `` {
		source = mountPoint
	}

	scheme, _ := stringutil.SplitPair(source, `:`)

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

	return mount, nil
}

func mountSummary(mount Mount) string {
	var mtype = fmt.Sprintf("%T", mount)
	mtype = strings.TrimPrefix(mtype, `*diecast.`)
	mtype = strings.TrimSuffix(mtype, `Mount`)
	mtype = strings.ToLower(mtype)

	return fmt.Sprintf("%s: %s -> %s", mtype, mount.GetMountPoint(), mount.GetTarget())
}

func IsSameMount(first Mount, second Mount) bool {
	if first != nil {
		if second != nil {
			if first.GetMountPoint() == second.GetMountPoint() {
				if first.GetTarget() == second.GetTarget() {
					return true
				}
			}
		}
	}

	return false
}

func IsHardStop(err error) bool {
	if err == MountHaltErr {
		return true
	} else if _, ok := err.(*url.Error); ok {
		return true
	} else if _, ok := err.(net.Error); ok {
		return true
	} else if log.ErrContains(err, `request canceled`) {
		return true
	} else if log.ErrContains(err, `x509:`) {
		return true
	} else if log.ErrHasPrefix(err, `dial `) {
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
