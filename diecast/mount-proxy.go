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

            //  return the file if it opened without fail OR if we aren't supposed to passthrough to the next mount
                if file, err := mount.Open(name); err == nil || !mount.Passthrough {
                    log.Debugf("MountProxy: mount[%d] '%s' responding authoritatively to %s", i, mount.MountPoint, name)

                    return file, err
                }
            }
        }
    }

    log.Debugf("MountProxy: fallback mount responding to %s", name)

    return self.Fallback.Open(name)
}