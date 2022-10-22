package diecast

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/ghetzel/go-stockutil/executil"
)

var SassIndentString = `    `
var callbackMap sync.Map
var DartSassBin = executil.Env(`DIECAST_SASS_BIN`, `sass`)

type SassRenderer struct {
	server   *Server
	prewrite PrewriteFunc
}

func (self *SassRenderer) ShouldPrerender() bool {
	return true
}

func (self *SassRenderer) SetPrewriteFunc(fn PrewriteFunc) {
	self.prewrite = fn
}

func (self *SassRenderer) SetServer(server *Server) {
	self.server = server
}

func (self *SassRenderer) Render(w http.ResponseWriter, req *http.Request, options RenderOptions) error {
	defer options.Input.Close()

	var sass = executil.Command(DartSassBin, `--stdin`)

	if _, err := io.Copy(sass, options.Input); err == nil {
		if output, err := ioutil.ReadAll(sass); err == nil {
			w.Header().Set(`Content-Type`, `text/css; charset=utf-8`)

			if fn := self.prewrite; fn != nil {
				fn(req)
			}

			_, err = w.Write(output)
			return err
		} else {
			return fmt.Errorf("sass error: %v", err)
		}
	} else {
		return fmt.Errorf("Cannot read render input: %v", err)
	}
}
