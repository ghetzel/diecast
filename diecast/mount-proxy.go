package diecast

import (
    "net/http"
    log "github.com/Sirupsen/logrus"
)

type MountProxy struct {
    http.FileSystem

    Mounts   []Mount
    Fallback http.FileSystem
}

func (self *MountProxy) Open(name string) (http.File, error) {
    if self.Mounts != nil {
        for i, mount := range self.Mounts {
            if mount.WillRespondTo(name) {
                log.Debugf("MountProxy: mount[%d] '%s' responding to %s", i, mount.MountPoint, name)

                return mount.Open(name)
            }
        }
    }

    return self.Fallback.Open(name)
}