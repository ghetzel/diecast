package diecast

import (
    "net/http"
    "os"
    "path"
    "strings"
    log "github.com/Sirupsen/logrus"
)

type Mount struct {
    http.FileSystem

    MountPoint string  `json:"mount"`
    Path       string  `json:"path"`
}

func (self *Mount) Initialize() error {
    if _, err := os.Stat(self.Path); err != nil {
        return err
    }

    log.Debugf("Initialize mount '%s' -> '%s'", self.MountPoint, self.Path)

    return nil
}

func (self *Mount) WillRespondTo(name string) bool {
    return strings.HasPrefix(name, self.MountPoint)
}

func (self *Mount) Open(name string) (http.File, error) {
    newPath := path.Join(strings.TrimSuffix(self.Path, `/`), strings.TrimPrefix(name, self.MountPoint))

    log.Debugf("Opening path '%s'", newPath)

    return os.Open(newPath)
}