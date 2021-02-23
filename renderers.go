package diecast

import (
	"fmt"
	"net/http"

	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

var renderers = make(map[string]Renderer)

func init() {
	// RegisterRenderer(`pdf`, new(TemplateRenderer))
	// RegisterRenderer(`image`, new(TemplateRenderer))
	// RegisterRenderer(`msoffice`, new(TemplateRenderer))
	// RegisterRenderer(`template`, new(TemplateRenderer))
	RegisterRenderer(``, new(PassthroughRenderer))
}

func RegisterRenderer(name string, renderer Renderer) {
	renderers[name] = renderer
}

type RendererConfig struct {
	Type    string                 `yaml:"type"`
	Options map[string]interface{} `yaml:"options"`
	Only    interface{}            `yaml:"only"`
	Except  interface{}            `yaml:"except"`
	Methods interface{}            `yaml:"methods"`
}

// Return whether the given request is eligible for rendering.
func (self *RendererConfig) ShouldApplyTo(req *http.Request) bool {
	return ShouldApplyTo(req, self.Except, self.Only, self.Methods)
}

// Return a typeutil.Variant containing the value at the named option key, or a fallback value.
func (self *RendererConfig) Option(name string, fallbacks ...interface{}) typeutil.Variant {
	return maputil.M(self.Options).Get(name, fallbacks...)
}

// =====================================================================================================================

// Render a retrieved file to the given response writer.
func (self *Server) serveHttpPhaseRender(ctx *Context, file http.File) error {
	// apply the first matching renderer from the config (if any)
	for _, rc := range self.Renderers {
		if rc.Type != `` {
			if rc.ShouldApplyTo(ctx.Request()) {
				if renderer, ok := renderers[rc.Type]; ok {
					return renderer.Render(ctx, file, &rc)
				}
			}
		} else {
			return fmt.Errorf("unrecognized renderer type %q", rc.Type)
		}
	}

	// fallback to just copying the retrieved data to the response directly
	return Passthrough(ctx, file, nil)
}
