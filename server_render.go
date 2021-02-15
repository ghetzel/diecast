package diecast

import (
	"io"
	"net/http"
)

var renderers = make(map[string]Renderer)

func init() {
	// RegisterRenderer(`template`, TemplateRenderer)
	// RegisterRenderer(`default`, PassthroughRenderer)
}

func RegisterRenderer(name string, renderer Renderer) {
	renderers[name] = renderer
}

// Render a retrieved file to the given response writer.
func (self *Server) Render(w http.ResponseWriter, req *http.Request, source http.File) error {
	_, err := io.Copy(w, source)
	return err
}
