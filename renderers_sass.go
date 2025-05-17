package diecast

import (
	"fmt"
	"io"
	"net/http"

	"github.com/ghetzel/go-stockutil/executil"
)

var SassIndentString = `    `
var DartSassBin = executil.Env(`DIECAST_SASS_BIN`, `sass`)

type SassRenderer struct {
	server   *Server
	prewrite PrewriteFunc
}

func (renderer *SassRenderer) ShouldPrerender() bool {
	return true
}

func (renderer *SassRenderer) SetPrewriteFunc(fn PrewriteFunc) {
	renderer.prewrite = fn
}

func (renderer *SassRenderer) SetServer(server *Server) {
	renderer.server = server
}

func (renderer *SassRenderer) Render(w http.ResponseWriter, req *http.Request, options RenderOptions) error {
	defer options.Input.Close()

	var sass = executil.Command(DartSassBin, `--stdin`)

	if _, err := io.Copy(sass, options.Input); err == nil {
		if output, err := io.ReadAll(sass); err == nil {
			w.Header().Set(`Content-Type`, `text/css; charset=utf-8`)

			if fn := renderer.prewrite; fn != nil {
				fn(req)
			}

			_, err = w.Write(output)
			return err
		} else {
			return fmt.Errorf("sass error: %v", err)
		}
	} else {
		return fmt.Errorf("cannot read render input: %v", err)
	}
}
