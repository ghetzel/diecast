package diecast

import (
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

type PassthroughRenderer struct{}

func Passthrough(r *RendererConfig) error {
	return new(PassthroughRenderer).Render(r)
}

func (self *PassthroughRenderer) Render(cfg *RendererConfig) error {
	if cfg != nil && cfg.Response != nil && cfg.Data != nil {
		_, err := io.Copy(cfg.Response, cfg.Data)
		return err
	} else {
		return nil
	}
}

type RendererConfig struct {
	Type     string                 `yaml:"type"`
	Options  map[string]interface{} `yaml:"options"`
	Only     interface{}            `yaml:"only"`
	Except   interface{}            `yaml:"except"`
	Methods  interface{}            `yaml:"methods"`
	Request  *http.Request          `yaml:"-"`
	Response http.ResponseWriter    `yaml:"-"`
	Data     io.ReadCloser          `yaml:"-"`
}

// Return whether the given request is eligible for rendering.
func (self *RendererConfig) ShouldApplyTo(req *http.Request) bool {
	return ShouldApplyTo(req, self.Except, self.Only, self.Methods)
}

// Return a typeutil.Variant containing the value at the named option key, or a fallback value.
func (self *RendererConfig) Option(name string, fallbacks ...interface{}) typeutil.Variant {
	return maputil.M(self.Options).Get(name, fallbacks...)
}

func (self RendererConfig) WithResponse(w http.ResponseWriter, req *http.Request, source io.ReadCloser) *RendererConfig {
	var cfg = self

	cfg.Response = w
	cfg.Request = req
	cfg.Data = source

	return &cfg
}

// =====================================================================================================================

// Render a retrieved file to the given response writer.
func (self *Server) Render(input *RendererConfig) error {
	var w = input.Response
	var req = input.Request
	var source = input.Data

	// apply the first matching renderer from the config (if any)
	for _, rc := range self.Renderers {
		if rc.Type != `` {
			if rc.ShouldApplyTo(req) {
				if renderer, ok := renderers[rc.Type]; ok {
					return renderer.Render(rc.WithResponse(w, req, source))
				}
			}
		} else {
			return fmt.Errorf("unrecognized renderer type %q", rc.Type)
		}
	}

	// fallback to just copying the retrieved data to the response directly
	return Passthrough(input)
}
