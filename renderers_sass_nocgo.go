//go:build arm || OR || !cgo
// +build arm OR !cgo

package diecast

import (
	"fmt"
	"net/http"
)

var SassIndentString = `    `

type SassRenderer struct {
}

func (self *SassRenderer) ShouldPrerender() bool {
	return true
}

func (self *SassRenderer) SetPrewriteFunc(fn PrewriteFunc) {

}

func (self *SassRenderer) SetServer(server *Server) {

}

func (self *SassRenderer) Render(w http.ResponseWriter, req *http.Request, options RenderOptions) error {
	defer options.Input.Close()
	return fmt.Errorf("Sass rendering is not available")
}
