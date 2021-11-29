package diecast

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

var rendererTypes = make(map[string]Renderer)
var renderersByMimeType = make(map[string]RendererConfig)
var renderersByGlob = make(map[string]RendererConfig)

func init() {
	// RegisterRendererType(`pdf`, new(TemplateRenderer))
	// RegisterRendererType(`image`, new(TemplateRenderer))
	// RegisterRendererType(`msoffice`, new(TemplateRenderer))
	RegisterRendererType(`template`, new(TemplateRenderer))
	RegisterRendererType(``, new(PassthroughRenderer))

	// setup default type handlers
	RegisterRendererByMIME(`text/html`, RendererConfig{
		Type: `template`,
		Methods: []string{
			http.MethodGet,
		},
	})
}

func RegisterRendererType(name string, renderer Renderer) {
	rendererTypes[name] = renderer
}

func RegisterRendererByGlob(fileglob string, cfg RendererConfig) {
	renderersByGlob[fileglob] = cfg
}

func RegisterRendererByMIME(mediaType string, cfg RendererConfig) {
	renderersByMimeType[strings.ToLower(mediaType)] = cfg
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

// Return a usable instance of the renderer that should be used (if any) for this configuration.
func (self *RendererConfig) RendererFor(ctx *Context) Renderer {
	if self.Type != `` {
		if ctx != nil && ctx.Request() != nil {
			if self.ShouldApplyTo(ctx.Request()) {
				return rendererTypes[self.Type]
			}
		} else {
			return rendererTypes[self.Type]
		}
	}

	return nil
}

// =====================================================================================================================

// Render a retrieved file to the given response writer.
func (self *Server) serveHttpPhaseRender(ctx *Context, file fs.File) error {
	// apply the first matching renderer from the config (if any)
	for _, rc := range self.Renderers {
		if renderer := rc.RendererFor(ctx); renderer != nil {
			ctx.Debugf("renderer: %T (config)", renderer)
			return renderer.Render(ctx, file, &rc)
		}
	}

	// try to find a renderer by glob matching the request path
	for pattern, rc := range renderersByGlob {
		if IsGlobMatch(ctx.Request().URL.Path, pattern) {
			if renderer := rc.RendererFor(ctx); renderer != nil {
				ctx.Debugf("renderer: %T (glob: %q)", renderer, pattern)
				return renderer.Render(ctx, file, &rc)
			}
		}
	}

	// try to work out a renderer based on the most recent MIME type hint
	if typeHint := ctx.TypeHint(); typeHint != `` {
		if rc, ok := renderersByMimeType[strings.ToLower(typeHint)]; ok {
			if renderer := rc.RendererFor(ctx); renderer != nil {
				ctx.Debugf("renderer: %T (mime: %q)", renderer, typeHint)
				return renderer.Render(ctx, file, &rc)
			}
		}
	}

	// try to find a renderer by glob matching the source path
	if stat, err := file.Stat(); err == nil {
		for pattern, rc := range renderersByGlob {
			if IsGlobMatch(stat.Name(), pattern) {
				if renderer := rc.RendererFor(ctx); renderer != nil {
					ctx.Debugf("renderer: %T (source glob: %q)", renderer, pattern)
					return renderer.Render(ctx, file, &rc)
				}
			}
		}
	}

	// fallback to just copying the retrieved data to the response directly
	return Passthrough(ctx, file, nil)
}
