package diecast

import (
	"net/http"
	"strings"
)

type MountProxy struct {
	http.FileSystem
	Server           *Server
	TemplatePatterns []string
	Mounts           []Mount
	Fallback         http.FileSystem
}

func (self *MountProxy) Open(name string) (http.File, error) {
	name = strings.TrimPrefix(name, self.Server.RoutePrefix)

	if mount := self.FindMountForEndpoint(name); mount != nil {
		//  return the file if it opened without fail OR if we aren't supposed to passthrough to the next mount
		if file, err := mount.Open(name); err == nil || !mount.Passthrough {
			return file, err
		}
	}

	file, err := self.Fallback.Open(name)

	if err == nil {
		log.Debugf("Static file found: %q", name)
	}

	return file, err
}

func (self *MountProxy) FindMountForEndpoint(endpointPath string) *Mount {
	if self.Mounts != nil {
		for i, mount := range self.Mounts {
			if mount.WillRespondTo(endpointPath) {
				log.Debugf("MountProxy: mount[%d] '%s' responding authoritatively to %s", i, mount.MountPoint, endpointPath)
				return &mount
			}
		}
	}

	return nil
}
