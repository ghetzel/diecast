// +build nocgo

package diecast

import (
	"io/ioutil"
	"net/http"

	"github.com/wellington/sass/compiler"
)

type SassRenderer struct {
	server *Server
}

func (self *SassRenderer) ShouldPrerender() bool {
	return true
}

func (self *SassRenderer) Render(w http.ResponseWriter, req *http.Request, options RenderOptions) error {
	defer options.Input.Close()

	if input, err := ioutil.ReadAll(options.Input); err == nil {
		if output, err := compiler.Compile(input); err == nil {
			w.Header().Set(`Content-Type`, `text/css; charset=utf-8`)
			_, err := w.Write(output)
			return err
		} else {
			return err
		}
	} else {
		return err
	}
}
