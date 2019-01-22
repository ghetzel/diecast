package diecast

import (
	"net/http"

	"github.com/wellington/go-libsass"
)

type SassRenderer struct {
	server *Server
}

func (self *SassRenderer) ShouldPrerender() bool {
	return true
}

func (self *SassRenderer) Render(w http.ResponseWriter, req *http.Request, options RenderOptions) error {
	defer options.Input.Close()

	if sass, err := libsass.New(w, options.Input, libsass.OutputStyle(libsass.EXPANDED_STYLE)); err == nil {
		w.Header().Set(`Content-Type`, `text/css; charset=utf-8`)

		return sass.Run()
	} else {
		return err
	}
}
