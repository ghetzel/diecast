package diecast

import (
	"bytes"
	"context"
	"fmt"
	"io"
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
	request *http.Request
	data    io.ReadCloser
}

func newRenderConfigFromRequest(req *http.Request, d io.ReadCloser) *RendererConfig {
	return &RendererConfig{
		data:    d,
		request: req,
	}
}

// Return whether the local request is eligible for renderering.
func (self *RendererConfig) ShouldApply() bool {
	if self.request == nil {
		return false
	} else {
		return self.ShouldApplyTo(self.request)
	}
}

// Return a copy of the local request.
func (self *RendererConfig) Request() *http.Request {
	if self.request == nil {
		return nil
	} else {
		return self.request.Clone(context.Background())
	}
}

func (self *RendererConfig) Data() io.ReadCloser {
	if self.data == nil {
		return io.NopCloser(bytes.NewBuffer(nil))
	} else {
		return self.data
	}
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
func (self *Server) Render(w http.ResponseWriter, input *RendererConfig) error {
	// apply the first matching renderer from the config (if any)
	for _, rc := range self.Renderers {
		if rc.Type != `` {
			if rc.ShouldApply() {
				if renderer, ok := renderers[rc.Type]; ok {
					return renderer.Render(w, input)
				}
			}
		} else {
			return fmt.Errorf("unrecognized renderer type %q", rc.Type)
		}
	}

	// fallback to just copying the retrieved data to the response directly
	return Passthrough(w, input)
}
