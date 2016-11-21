package diecast

import (
	"net/http"
	"strings"
)

type MountProxy struct {
	http.FileSystem
	RoutePrefix      string
	TemplatePatterns []string
	Mounts           []Mount
	Fallback         http.FileSystem
}

func (self *MountProxy) Open(name string) (http.File, error) {
	name = strings.TrimPrefix(name, self.RoutePrefix)

	if mount := self.FindMountForEndpoint(name); mount != nil {
		//  return the file if it opened without fail OR if we aren't supposed to passthrough to the next mount
		if file, err := mount.Open(name); err == nil || !mount.Passthrough {
			return file, err
		}
	}

	file, err := self.Fallback.Open(name)

	if err == nil {
		log.Debugf("Static file %q found", name)
	} else {
		log.Debugf("Static file %q not found: %v", name, err)
	}

	return file, err
}

func (self *MountProxy) FindMountForEndpoint(endpointPath string) *Mount {
	endpointPath = strings.TrimPrefix(endpointPath, self.RoutePrefix)

	for _, mount := range self.Mounts {
		if mount.WillRespondTo(endpointPath) {
			log.Debugf("mount[%s] Handling %q", mount.MountPoint, endpointPath)
			return &mount
		}
	}

	return nil
}
